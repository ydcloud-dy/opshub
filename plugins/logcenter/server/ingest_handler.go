package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/loggingest"
	"github.com/ydcloud-dy/opshub/pkg/response"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	"gorm.io/gorm"
)

type ingestComponentResult struct {
	loggingest.ComponentStatus
	Reachable bool `json:"reachable"`
}

type ingestQueueResult struct {
	Enabled           bool   `json:"enabled"`
	Status            string `json:"status"`
	Reachable         bool   `json:"reachable"`
	Topic             string `json:"topic,omitempty"`
	ConsumerGroup     string `json:"consumerGroup,omitempty"`
	BrokerCount       int    `json:"brokerCount,omitempty"`
	Lag               int64  `json:"lag,omitempty"`
	DeadletterBatches uint64 `json:"deadletterBatches,omitempty"`
	LastError         string `json:"lastError,omitempty"`
}

type ingestStorageResult struct {
	ID                     uint       `json:"id,omitempty"`
	Name                   string     `json:"name,omitempty"`
	Status                 string     `json:"status"`
	Reachable              bool       `json:"reachable"`
	LastTestAt             *time.Time `json:"lastTestAt,omitempty"`
	LastError              string     `json:"lastError,omitempty"`
	InitializedAt          *time.Time `json:"initializedAt,omitempty"`
	RetentionStatus        string     `json:"retentionStatus,omitempty"`
	RetentionError         string     `json:"retentionError,omitempty"`
	ExpiredParts           int64      `json:"expiredParts"`
	TTLLagSeconds          int64      `json:"ttlLagSeconds"`
	TTLMergeActive         bool       `json:"ttlMergeActive"`
	TTLMergeProgress       float64    `json:"ttlMergeProgress"`
	TTLMergeTimeoutSeconds int        `json:"ttlMergeTimeoutSeconds"`
}

type ingestReadinessCheck struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	Description    string `json:"description"`
	Recommendation string `json:"recommendation,omitempty"`
}

type ingestReadinessSummary struct {
	Passed   int `json:"passed"`
	Warnings int `json:"warnings"`
	Failed   int `json:"failed"`
	Total    int `json:"total"`
}

func (h *Handler) GetIngestStatus(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以查看日志采集链路") {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	gatewayURL := strings.TrimRight(firstNonEmpty(os.Getenv("OPSHUB_LOG_GATEWAY_STATUS_URL"), "http://127.0.0.1:9880"), "/")
	writerURL := strings.TrimRight(firstNonEmpty(os.Getenv("OPSHUB_LOG_WRITER_STATUS_URL"), "http://127.0.0.1:9881"), "/")
	var gateway, writer ingestComponentResult
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	go func() {
		defer waitGroup.Done()
		gateway = fetchIngestComponent(ctx, gatewayURL+"/status", "log-gateway")
	}()
	go func() {
		defer waitGroup.Done()
		writer = fetchIngestComponent(ctx, writerURL+"/status", "log-writer")
	}()
	storage := h.loadIngestStorageStatus(ctx)
	waitGroup.Wait()
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
	queue := ingestQueueResult{
		Enabled: queueEnabled, Status: queueStatus, Reachable: queueReachable,
		Topic: firstNonEmpty(gateway.QueueTopic, writer.QueueTopic), ConsumerGroup: writer.ConsumerGroup,
		BrokerCount: maxInt(gateway.BrokerCount, writer.BrokerCount), Lag: writer.QueueLag,
		DeadletterBatches: writer.DeadletterBatches,
		LastError:         firstNonEmpty(writer.QueueLastError, gateway.QueueLastError, gateway.LastError, writer.LastError),
	}
	publicGatewayURL := resolveAgentGatewayURL(c)
	checks := buildIngestReadiness(gateway, writer, queue, storage, publicGatewayURL)
	response.Success(c, gin.H{
		"mode": mode, "gateway": gateway, "queue": queue, "writer": writer, "storage": storage,
		"gatewayUrl": gatewayURL, "writerUrl": writerURL, "publicGatewayUrl": publicGatewayURL,
		"readiness": checks, "readinessSummary": summarizeIngestReadiness(checks), "checkedAt": time.Now(),
	})
}

func (h *Handler) loadIngestStorageStatus(ctx context.Context) ingestStorageResult {
	result := ingestStorageResult{Status: "unconfigured"}
	if h == nil || h.db == nil {
		result.Status = "error"
		result.LastError = "日志存储数据库未初始化"
		return result
	}
	var storage logmodel.StorageCluster
	if err := h.db.WithContext(ctx).Where("enabled = ?", true).Order("is_primary DESC, id ASC").First(&storage).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			result.Status = "error"
			result.LastError = err.Error()
		}
		return result
	}
	result = ingestStorageResult{
		ID: storage.ID, Name: storage.Name, Status: storage.Status,
		LastTestAt: storage.LastTestAt, LastError: storage.LastError, InitializedAt: storage.InitializedAt,
	}
	password, err := decryptStorageSecret(storage.PasswordEncrypted)
	if err == nil && h.clickhouse != nil {
		err = h.clickhouse.Ping(ctx, storage, password)
	}
	if err != nil {
		result.Status = "error"
		result.LastError = err.Error()
		return result
	}
	if h.clickhouse == nil {
		result.Status = "error"
		result.LastError = "ClickHouse 客户端未初始化"
		return result
	}
	result.Status = "healthy"
	result.Reachable = true
	result.LastError = ""
	retention, retentionErr := h.clickhouse.RetentionHealth(ctx, storage, password)
	if retentionErr != nil {
		result.RetentionStatus = "error"
		result.RetentionError = retentionErr.Error()
		return result
	}
	result.RetentionStatus = retention.Status
	result.ExpiredParts = retention.ExpiredParts
	result.TTLLagSeconds = retention.TTLLagSeconds
	result.TTLMergeActive = retention.TTLMergeActive
	result.TTLMergeProgress = retention.TTLMergeProgress
	result.TTLMergeTimeoutSeconds = retention.TTLMergeTimeoutSeconds
	return result
}

func buildIngestReadiness(gateway, writer ingestComponentResult, queue ingestQueueResult, storage ingestStorageResult, publicGatewayURL string) []ingestReadinessCheck {
	checks := []ingestReadinessCheck{
		componentReadiness("gateway", "Log Gateway", gateway),
		componentReadiness("writer", "Log Writer", writer),
		storageReadiness(storage),
		retentionReadiness(storage),
		queueReadiness(queue),
		publicGatewayReadiness(publicGatewayURL),
		secretReadiness(),
		collectorImageReadiness(strings.TrimSpace(os.Getenv("OPSHUB_LOG_AGENT_IMAGE"))),
	}
	return checks
}

func retentionReadiness(storage ingestStorageResult) ingestReadinessCheck {
	check := ingestReadinessCheck{ID: "retention", Title: "日志保留清理"}
	if storage.ID == 0 || !storage.Reachable {
		check.Status = "failed"
		check.Description = "无法检查日志保留清理状态"
		check.Recommendation = firstNonEmpty(storage.LastError, "先恢复 ClickHouse 日志存储")
		return check
	}
	if storage.RetentionStatus == "error" || storage.RetentionStatus == "" {
		check.Status = "failed"
		check.Description = "读取 ClickHouse TTL 状态失败"
		check.Recommendation = firstNonEmpty(storage.RetentionError, "检查 system.parts 和 system.merges 查询权限")
		return check
	}
	description := fmt.Sprintf("过期 Part %d 个，最老清理延迟 %s", storage.ExpiredParts, formatRetentionLag(storage.TTLLagSeconds))
	if storage.TTLMergeActive {
		description += fmt.Sprintf("，TTL Merge %.0f%%", storage.TTLMergeProgress*100)
	}
	check.Description = description
	switch storage.RetentionStatus {
	case "critical":
		check.Status = "failed"
		check.Recommendation = "TTL 清理延迟超过调度周期 6 倍，请检查 Merge 队列、磁盘和后台线程"
	case "warning":
		check.Status = "warning"
		check.Recommendation = "TTL 清理存在积压，持续观察 Merge 进度和磁盘使用率"
	default:
		check.Status = "passed"
	}
	return check
}

func formatRetentionLag(seconds int64) string {
	if seconds <= 0 {
		return "0 分钟"
	}
	if seconds < 3600 {
		return fmt.Sprintf("%d 分钟", (seconds+59)/60)
	}
	return fmt.Sprintf("%.1f 小时", float64(seconds)/3600)
}

func componentReadiness(id, title string, component ingestComponentResult) ingestReadinessCheck {
	if !component.Reachable {
		return ingestReadinessCheck{ID: id, Title: title, Status: "failed", Description: "控制面无法访问该服务", Recommendation: firstNonEmpty(component.LastError, "检查容器、Service 和状态地址配置")}
	}
	if component.Status != "healthy" {
		return ingestReadinessCheck{ID: id, Title: title, Status: "warning", Description: "服务可访问，但当前处于降级状态", Recommendation: firstNonEmpty(component.LastError, component.QueueLastError, "检查失败批次、队列连接和运行日志")}
	}
	return ingestReadinessCheck{ID: id, Title: title, Status: "passed", Description: "服务运行正常，状态接口可访问"}
}

func storageReadiness(storage ingestStorageResult) ingestReadinessCheck {
	if storage.ID == 0 {
		return ingestReadinessCheck{ID: "storage", Title: "ClickHouse 日志存储", Status: "failed", Description: "尚未配置可用的日志存储", Recommendation: "前往日志库添加并初始化 ClickHouse"}
	}
	if storage.InitializedAt == nil {
		return ingestReadinessCheck{ID: "storage", Title: "ClickHouse 日志存储", Status: "failed", Description: "存储已配置，但日志表尚未初始化", Recommendation: "在日志库中执行初始化，操作不会清空已有数据"}
	}
	if !storage.Reachable {
		return ingestReadinessCheck{ID: "storage", Title: "ClickHouse 日志存储", Status: "failed", Description: "实时连接检查失败", Recommendation: firstNonEmpty(storage.LastError, "检查 ClickHouse 地址、账号和网络")}
	}
	return ingestReadinessCheck{ID: "storage", Title: "ClickHouse 日志存储", Status: "passed", Description: "实时连接成功，日志表已初始化"}
}

func queueReadiness(queue ingestQueueResult) ingestReadinessCheck {
	if !queue.Enabled {
		return ingestReadinessCheck{ID: "queue", Title: "可靠消息队列", Status: "warning", Description: "当前为直写模式，ClickHouse 故障时会对 Agent 产生背压", Recommendation: "生产大规模环境建议启用 Redpanda 或 Kafka"}
	}
	if !queue.Reachable {
		return ingestReadinessCheck{ID: "queue", Title: "可靠消息队列", Status: "failed", Description: "Kafka/Redpanda 连接或消费状态异常", Recommendation: firstNonEmpty(queue.LastError, "检查 Broker、Topic 和 Consumer Group")}
	}
	return ingestReadinessCheck{ID: "queue", Title: "可靠消息队列", Status: "passed", Description: fmt.Sprintf("队列连接正常，当前消费积压 %d", queue.Lag)}
}

func publicGatewayReadiness(value string) ingestReadinessCheck {
	check := ingestReadinessCheck{ID: "public-gateway", Title: "Agent 接入地址"}
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		check.Status = "failed"
		check.Description = "没有解析出有效的 Agent 接入地址"
		check.Recommendation = "配置 OPSHUB_LOG_GATEWAY_PUBLIC_URL 或平台外部访问地址"
		return check
	}
	host := strings.ToLower(parsed.Hostname())
	ip := net.ParseIP(host)
	if host == "localhost" || (ip != nil && (ip.IsLoopback() || ip.IsUnspecified())) {
		check.Status = "failed"
		check.Description = "Agent 接入地址仍指向本机地址"
		check.Recommendation = "配置采集主机和 Kubernetes 集群可访问的域名或 IP"
		return check
	}
	check.Status = "passed"
	check.Description = strings.TrimRight(value, "/")
	return check
}

func secretReadiness() ingestReadinessCheck {
	values := map[string]string{
		"日志写入 Token": firstNonEmpty(os.Getenv("OPSHUB_LOG_AGENT_INGEST_TOKEN"), os.Getenv("OPSHUB_LOG_INGEST_TOKEN"), os.Getenv("OPSHUB_LOG_INGEST_TEST_TOKEN")),
		"日志凭据加密密钥":   os.Getenv("OPSHUB_LOGCENTER_ENCRYPTION_KEY"),
	}
	missing := make([]string, 0)
	defaults := make([]string, 0)
	for name, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			missing = append(missing, name)
			continue
		}
		lower := strings.ToLower(value)
		if strings.Contains(lower, "change-in-production") || lower == "opshub-local-ingest-token" {
			defaults = append(defaults, name)
		}
	}
	if len(missing) > 0 {
		return ingestReadinessCheck{ID: "secrets", Title: "生产密钥", Status: "failed", Description: strings.Join(missing, "、") + "未配置", Recommendation: "生成独立随机值并通过环境变量或 Kubernetes Secret 注入"}
	}
	if len(defaults) > 0 {
		return ingestReadinessCheck{ID: "secrets", Title: "生产密钥", Status: "warning", Description: strings.Join(defaults, "、") + "仍在使用默认值", Recommendation: "正式接入采集器前更换默认密钥并滚动更新相关服务"}
	}
	return ingestReadinessCheck{ID: "secrets", Title: "生产密钥", Status: "passed", Description: "日志 Token 与凭据加密密钥已显式配置"}
}

func collectorImageReadiness(image string) ingestReadinessCheck {
	check := ingestReadinessCheck{ID: "collector-image", Title: "Collector 镜像"}
	if image == "" {
		check.Status = "failed"
		check.Description = "未配置 Kubernetes Collector 镜像"
		check.Recommendation = "配置与当前控制面兼容的 opshub-log-agent 镜像"
		return check
	}
	lower := strings.ToLower(image)
	if strings.HasSuffix(lower, ":latest") || (!strings.Contains(image[strings.LastIndex(image, "/")+1:], ":") && !strings.Contains(lower, "@sha256:")) {
		check.Status = "warning"
		check.Description = image
		check.Recommendation = "生产环境请使用不可变版本标签或镜像摘要"
		return check
	}
	check.Status = "passed"
	check.Description = image
	return check
}

func summarizeIngestReadiness(checks []ingestReadinessCheck) ingestReadinessSummary {
	summary := ingestReadinessSummary{Total: len(checks)}
	for _, check := range checks {
		switch check.Status {
		case "passed":
			summary.Passed++
		case "warning":
			summary.Warnings++
		case "failed":
			summary.Failed++
		}
	}
	return summary
}

func (h *Handler) TestIngest(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以写入采集链路测试日志") {
		return
	}
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
