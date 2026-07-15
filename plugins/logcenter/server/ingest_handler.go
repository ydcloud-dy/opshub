package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/loggingest"
	"github.com/ydcloud-dy/opshub/pkg/response"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
)

type ingestComponentResult struct {
	loggingest.ComponentStatus
	Reachable bool `json:"reachable"`
}

func (h *Handler) GetIngestStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	gatewayURL := strings.TrimRight(firstNonEmpty(os.Getenv("OPSHUB_LOG_GATEWAY_STATUS_URL"), "http://127.0.0.1:9880"), "/")
	writerURL := strings.TrimRight(firstNonEmpty(os.Getenv("OPSHUB_LOG_WRITER_STATUS_URL"), "http://127.0.0.1:9881"), "/")
	gateway := fetchIngestComponent(ctx, gatewayURL+"/status", "log-gateway")
	writer := fetchIngestComponent(ctx, writerURL+"/status", "log-writer")
	mode := strings.ToLower(firstNonEmpty(gateway.QueueMode, writer.QueueMode, os.Getenv("OPSHUB_LOG_QUEUE_MODE"), "direct"))
	if mode == "redpanda" {
		mode = "kafka"
	}
	queueEnabled := mode == "kafka"
	queueReachable := true
	queueStatus := "bypassed"
	if queueEnabled {
		queueReachable = gateway.Reachable && writer.Reachable && gateway.QueueHealthy && writer.QueueHealthy
		queueStatus = "healthy"
		if !queueReachable {
			queueStatus = "degraded"
		}
	}
	queue := gin.H{
		"enabled": queueEnabled, "status": queueStatus, "reachable": queueReachable,
		"topic":         firstNonEmpty(gateway.QueueTopic, writer.QueueTopic),
		"consumerGroup": writer.ConsumerGroup, "brokerCount": maxInt(gateway.BrokerCount, writer.BrokerCount),
		"lag": writer.QueueLag, "deadletterBatches": writer.DeadletterBatches,
		"lastError": firstNonEmpty(writer.QueueLastError, gateway.QueueLastError, gateway.LastError, writer.LastError),
	}

	var storage logmodel.StorageCluster
	storageStatus := gin.H{"status": "unconfigured", "reachable": false}
	if err := h.db.Where("enabled = ?", true).Order("is_primary DESC, id ASC").First(&storage).Error; err == nil {
		prepareStorageForResponse(&storage)
		storageStatus = gin.H{
			"id": storage.ID, "name": storage.Name, "status": storage.Status,
			"reachable": storage.Status == "healthy", "lastTestAt": storage.LastTestAt,
			"lastError": storage.LastError, "initializedAt": storage.InitializedAt,
		}
	}
	response.Success(c, gin.H{
		"mode":       mode,
		"gateway":    gateway,
		"queue":      queue,
		"writer":     writer,
		"storage":    storageStatus,
		"gatewayUrl": gatewayURL,
		"writerUrl":  writerURL,
		"checkedAt":  time.Now(),
	})
}

func (h *Handler) TestIngest(c *gin.Context) {
	gatewayURL := strings.TrimRight(firstNonEmpty(os.Getenv("OPSHUB_LOG_GATEWAY_STATUS_URL"), "http://127.0.0.1:9880"), "/")
	token := resolveIngestTestToken(gatewayURL)
	if token == "" {
		response.ErrorCode(c, http.StatusServiceUnavailable, "未配置日志采集测试 Token，无法执行链路测试")
		return
	}
	var req struct {
		Message string `json:"message"`
		Level   string `json:"level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = "OpsHub 日志采集链路测试成功"
	}
	now := time.Now()
	batch := loggingest.LogBatch{
		BatchID: uuidV4(), AgentID: "opshub-control-plane", SourceType: "system",
		AssetType: "system", AssetID: 1, Environment: "test", Service: "opshub-logcenter",
		Stream: "stdout", SequenceStart: uint64(now.UnixNano()), SequenceEnd: uint64(now.UnixNano()),
		Records: []loggingest.LogRecord{{
			Sequence: uint64(now.UnixNano()), TimestampUnixNano: now.UnixNano(),
			ObservedTimestampUnixNano: now.UnixNano(), Body: message,
			SeverityText: firstNonEmpty(req.Level, "INFO"),
			Attributes:   map[string]string{"test": "true", "origin": "logcenter-ui"},
		}},
	}
	raw, err := json.Marshal(batch)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "生成测试日志失败: "+err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayURL+"/api/v1/logs/batches", bytes.NewReader(raw))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建测试请求失败: "+err.Error())
		return
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	result, err := http.DefaultClient.Do(request)
	if err != nil {
		response.ErrorCode(c, http.StatusBadGateway, "Log Gateway 不可用: "+err.Error())
		return
	}
	defer result.Body.Close()
	var ack loggingest.IngestAck
	if err := json.NewDecoder(io.LimitReader(result.Body, 1024*1024)).Decode(&ack); err != nil {
		response.ErrorCode(c, http.StatusBadGateway, "解析采集确认失败: "+err.Error())
		return
	}
	if result.StatusCode < 200 || result.StatusCode >= 300 || ack.ErrorCode != "" {
		response.ErrorCode(c, http.StatusBadGateway, firstNonEmpty(ack.ErrorMessage, fmt.Sprintf("Log Gateway 返回 %d", result.StatusCode)))
		return
	}
	response.Success(c, gin.H{"ack": ack, "message": message, "timestamp": now})
}

func resolveIngestTestToken(gatewayURL string) string {
	if token := strings.TrimSpace(os.Getenv("OPSHUB_LOG_INGEST_TEST_TOKEN")); token != "" {
		return token
	}
	if token := strings.TrimSpace(os.Getenv("OPSHUB_LOG_INGEST_TOKEN")); token != "" {
		return token
	}
	parsed, err := url.Parse(gatewayURL)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()
	if strings.EqualFold(host, "localhost") || net.ParseIP(host).IsLoopback() {
		return "opshub-local-ingest-token"
	}
	return ""
}

func fetchIngestComponent(ctx context.Context, target, name string) ingestComponentResult {
	result := ingestComponentResult{ComponentStatus: loggingest.ComponentStatus{Name: name, Status: "unreachable"}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		result.LastError = err.Error()
		return result
	}
	httpResult, err := http.DefaultClient.Do(request)
	if err != nil {
		result.LastError = err.Error()
		return result
	}
	defer httpResult.Body.Close()
	if httpResult.StatusCode < 200 || httpResult.StatusCode >= 300 {
		result.LastError = fmt.Sprintf("HTTP %d", httpResult.StatusCode)
		return result
	}
	if err := json.NewDecoder(io.LimitReader(httpResult.Body, 1024*1024)).Decode(&result.ComponentStatus); err != nil {
		result.LastError = err.Error()
		return result
	}
	result.Reachable = true
	return result
}

func uuidV4() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return fmt.Sprintf("00000000-0000-4000-8000-%012x", time.Now().UnixNano()&0xffffffffffff)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(value)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}
