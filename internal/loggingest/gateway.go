package loggingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type GatewayConfig struct {
	WriterURL           string
	WriterToken         string
	QueueMode           string
	QueueTopic          string
	BrokerCount         int
	Publisher           BatchPublisher
	AgentTokens         []string
	MaxBodyBytes        int64
	RequestTimeout      time.Duration
	RatePerSecond       int
	BurstRecords        int
	GlobalRatePerSecond int
	GlobalBurstRecords  int
	MaxInflight         int
	Limits              Limits
	HTTPAddress         string
	GRPCAddress         string
}

type BatchPublisher interface {
	Publish(ctx context.Context, batch LogBatch) (IngestAck, error)
	Close()
}

type agentRate struct {
	lastRefill time.Time
	tokens     float64
}

type Gateway struct {
	config          GatewayConfig
	client          *http.Client
	startedAt       time.Time
	tokens          map[string]struct{}
	rateMu          sync.Mutex
	rates           map[string]*agentRate
	globalRate      *agentRate
	inflightSlots   chan struct{}
	inflight        atomic.Int64
	queueHealthy    atomic.Bool
	acceptedBatches atomic.Uint64
	acceptedRecords atomic.Uint64
	rejectedBatches atomic.Uint64
	failedBatches   atomic.Uint64
	publishNanos    atomic.Uint64
	publishCount    atomic.Uint64
	lastMu          sync.RWMutex
	lastSuccessAt   *time.Time
	lastErrorAt     *time.Time
	lastError       string
}

func NewGateway(config GatewayConfig) *Gateway {
	if config.MaxBodyBytes <= 0 {
		config.MaxBodyBytes = 2 * 1024 * 1024
	}
	if config.RequestTimeout <= 0 {
		config.RequestTimeout = 20 * time.Second
	}
	if config.RatePerSecond <= 0 {
		config.RatePerSecond = 5000
	}
	if config.BurstRecords <= 0 {
		config.BurstRecords = 10000
	}
	if config.GlobalRatePerSecond <= 0 {
		config.GlobalRatePerSecond = 50000
	}
	if config.GlobalBurstRecords <= 0 {
		config.GlobalBurstRecords = 100000
	}
	if config.MaxInflight <= 0 {
		config.MaxInflight = 256
	}
	config.QueueMode = strings.ToLower(strings.TrimSpace(config.QueueMode))
	if config.QueueMode == "" {
		config.QueueMode = "direct"
	}
	if config.Limits.MaxBatchRecords <= 0 {
		config.Limits = DefaultLimits()
	}
	tokens := make(map[string]struct{}, len(config.AgentTokens))
	for _, token := range config.AgentTokens {
		if token = strings.TrimSpace(token); token != "" {
			tokens[token] = struct{}{}
		}
	}
	gateway := &Gateway{
		config: config, client: &http.Client{Timeout: config.RequestTimeout}, startedAt: time.Now(),
		tokens: tokens, rates: make(map[string]*agentRate),
		globalRate:    &agentRate{lastRefill: time.Now(), tokens: float64(config.GlobalBurstRecords)},
		inflightSlots: make(chan struct{}, config.MaxInflight),
	}
	gateway.queueHealthy.Store(true)
	return gateway
}

func (g *Gateway) Submit(ctx context.Context, token string, batch LogBatch) IngestAck {
	if !g.authorized(token) {
		g.rejectedBatches.Add(1)
		return errorAck(batch.BatchID, "UNAUTHORIZED", "Agent Token 无效", 0)
	}
	if err := ValidateBatch(batch, g.config.Limits); err != nil {
		g.rejectedBatches.Add(1)
		return errorAck(batch.BatchID, "INVALID_BATCH", err.Error(), 0)
	}
	if !g.allow(batch.AgentID, len(batch.Records)) {
		g.rejectedBatches.Add(1)
		return errorAck(batch.BatchID, "RATE_LIMITED", "Agent 或 Gateway 全局上传速率超过限制", 1000)
	}
	select {
	case g.inflightSlots <- struct{}{}:
		g.inflight.Add(1)
		defer func() {
			<-g.inflightSlots
			g.inflight.Add(-1)
		}()
	default:
		g.rejectedBatches.Add(1)
		return errorAck(batch.BatchID, "GATEWAY_OVERLOADED", "Gateway 并发请求已达到上限", 500)
	}
	publishStarted := time.Now()
	ack, err := g.publish(ctx, batch)
	g.publishNanos.Add(uint64(time.Since(publishStarted)))
	g.publishCount.Add(1)
	if err != nil {
		g.failedBatches.Add(1)
		g.queueHealthy.Store(false)
		now := time.Now()
		g.lastMu.Lock()
		g.lastErrorAt = &now
		g.lastError = err.Error()
		g.lastMu.Unlock()
		return errorAck(batch.BatchID, "WRITER_UNAVAILABLE", err.Error(), 1000)
	}
	if ack.ErrorCode != "" {
		g.failedBatches.Add(1)
		g.queueHealthy.Store(false)
		return ack
	}
	g.queueHealthy.Store(true)
	g.acceptedBatches.Add(1)
	g.acceptedRecords.Add(uint64(len(batch.Records)))
	now := time.Now()
	g.lastMu.Lock()
	g.lastSuccessAt = &now
	g.lastError = ""
	g.lastMu.Unlock()
	return ack
}

func (g *Gateway) Status() ComponentStatus {
	g.lastMu.RLock()
	defer g.lastMu.RUnlock()
	status := "healthy"
	if !g.queueHealthy.Load() {
		status = "degraded"
	}
	return ComponentStatus{
		Name: "log-gateway", InstanceID: componentInstanceID(), Status: status,
		QueueMode: g.config.QueueMode, QueueTopic: g.config.QueueTopic, BrokerCount: g.config.BrokerCount,
		QueueHealthy: g.queueHealthy.Load(), StartedAt: g.startedAt,
		UptimeSeconds: int64(time.Since(g.startedAt).Seconds()), WriterURL: g.config.WriterURL,
		HTTPAddress: g.config.HTTPAddress, GRPCAddress: g.config.GRPCAddress,
		AcceptedBatches: g.acceptedBatches.Load(), AcceptedRecords: g.acceptedRecords.Load(),
		RejectedBatches: g.rejectedBatches.Load(), FailedBatches: g.failedBatches.Load(),
		Inflight: g.inflight.Load(), InflightLimit: g.config.MaxInflight,
		PublishLatencyMS: averageMilliseconds(g.publishNanos.Load(), g.publishCount.Load()),
		LastSuccessAt:    g.lastSuccessAt, LastErrorAt: g.lastErrorAt, LastError: g.lastError,
	}
}

func (g *Gateway) Close() {
	if g.config.Publisher != nil {
		g.config.Publisher.Close()
	}
}

func (g *Gateway) HTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(writer http.ResponseWriter, _ *http.Request) {
		status := g.Status()
		httpStatus := http.StatusOK
		if status.Status != "healthy" {
			httpStatus = http.StatusServiceUnavailable
		}
		writeJSON(writer, httpStatus, map[string]string{"status": status.Status})
	})
	mux.HandleFunc("/status", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, g.Status())
	})
	mux.HandleFunc("/metrics", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; version=0.0.4")
		status := g.Status()
		_, _ = fmt.Fprintf(writer, "opshub_log_gateway_accepted_batches_total %d\n", status.AcceptedBatches)
		_, _ = fmt.Fprintf(writer, "opshub_log_gateway_accepted_records_total %d\n", status.AcceptedRecords)
		_, _ = fmt.Fprintf(writer, "opshub_log_gateway_rejected_batches_total %d\n", status.RejectedBatches)
		_, _ = fmt.Fprintf(writer, "opshub_log_gateway_failed_batches_total %d\n", status.FailedBatches)
		_, _ = fmt.Fprintf(writer, "opshub_log_gateway_inflight %d\n", status.Inflight)
		_, _ = fmt.Fprintf(writer, "opshub_log_gateway_inflight_limit %d\n", status.InflightLimit)
		_, _ = fmt.Fprintf(writer, "opshub_log_gateway_publish_latency_ms %.3f\n", status.PublishLatencyMS)
		_, _ = fmt.Fprintf(writer, "opshub_log_gateway_queue_healthy %d\n", boolMetric(status.QueueHealthy))
	})
	mux.HandleFunc("/api/v1/logs/batches", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		request.Body = http.MaxBytesReader(writer, request.Body, g.config.MaxBodyBytes)
		var batch LogBatch
		if err := json.NewDecoder(request.Body).Decode(&batch); err != nil {
			writeJSON(writer, http.StatusBadRequest, errorAck(batch.BatchID, "INVALID_JSON", err.Error(), 0))
			return
		}
		ack := g.Submit(request.Context(), bearerToken(request), batch)
		status := http.StatusOK
		if ack.ErrorCode != "" {
			status = http.StatusBadRequest
			if ack.ErrorCode == "UNAUTHORIZED" {
				status = http.StatusUnauthorized
			} else if ack.RetryAfterMS > 0 {
				status = http.StatusServiceUnavailable
			}
		}
		writeJSON(writer, status, ack)
	})
	return mux
}

func (g *Gateway) authorized(token string) bool {
	if len(g.tokens) == 0 {
		return false
	}
	_, ok := g.tokens[strings.TrimSpace(token)]
	return ok
}

func (g *Gateway) allow(agentID string, records int) bool {
	now := time.Now()
	g.rateMu.Lock()
	defer g.rateMu.Unlock()
	state := g.rates[agentID]
	if state == nil {
		state = &agentRate{lastRefill: now, tokens: float64(g.config.BurstRecords)}
		g.rates[agentID] = state
	}
	refillRate(state, now, g.config.RatePerSecond, g.config.BurstRecords)
	refillRate(g.globalRate, now, g.config.GlobalRatePerSecond, g.config.GlobalBurstRecords)
	if state.tokens < float64(records) || g.globalRate.tokens < float64(records) {
		return false
	}
	state.tokens -= float64(records)
	g.globalRate.tokens -= float64(records)
	if len(g.rates) > 10000 {
		for key, rate := range g.rates {
			if now.Sub(rate.lastRefill) > 10*time.Minute {
				delete(g.rates, key)
			}
		}
	}
	return true
}

func refillRate(state *agentRate, now time.Time, rate, burst int) {
	elapsed := now.Sub(state.lastRefill).Seconds()
	state.tokens += elapsed * float64(rate)
	if state.tokens > float64(burst) {
		state.tokens = float64(burst)
	}
	state.lastRefill = now
}

func (g *Gateway) publish(ctx context.Context, batch LogBatch) (IngestAck, error) {
	if g.config.Publisher != nil {
		return g.config.Publisher.Publish(ctx, batch)
	}
	return g.forward(ctx, batch)
}

func (g *Gateway) forward(ctx context.Context, batch LogBatch) (IngestAck, error) {
	raw, err := json.Marshal(batch)
	if err != nil {
		return IngestAck{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(g.config.WriterURL, "/")+"/internal/v1/write", bytes.NewReader(raw))
	if err != nil {
		return IngestAck{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+g.config.WriterToken)
	response, err := g.client.Do(request)
	if err != nil {
		return IngestAck{}, err
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, 1024*1024)
	var ack IngestAck
	if err := json.NewDecoder(limited).Decode(&ack); err != nil {
		return ack, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ack, fmt.Errorf("Writer 返回 %d: %s", response.StatusCode, ack.ErrorMessage)
	}
	return ack, nil
}

func bearerToken(request *http.Request) string {
	if token := strings.TrimSpace(request.Header.Get("X-OpsHub-Agent-Token")); token != "" {
		return token
	}
	value := strings.TrimSpace(request.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return ""
}

func writeJSON(writer http.ResponseWriter, status int, value interface{}) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
