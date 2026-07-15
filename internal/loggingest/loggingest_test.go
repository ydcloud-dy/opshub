package loggingest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
)

const testBatchID = "a3d8af66-967f-4e0d-8ac7-94f79c977d62"

type testSink struct {
	calls     atomic.Int32
	failures  atomic.Int32
	started   chan struct{}
	release   chan struct{}
	startOnce sync.Once
}

func (s *testSink) WriteBatch(_ context.Context, _ LogBatch) error {
	s.calls.Add(1)
	if s.started != nil {
		s.startOnce.Do(func() { close(s.started) })
	}
	if s.release != nil {
		<-s.release
	}
	if s.failures.Add(-1) >= 0 {
		return errors.New("temporary failure")
	}
	return nil
}

func validBatch() LogBatch {
	return LogBatch{
		BatchID: testBatchID, AgentID: "agent-1", AssetType: "host", AssetID: 1, HostID: 1,
		SequenceStart: 1, SequenceEnd: 1,
		Records: []LogRecord{{Sequence: 1, TimestampUnixNano: time.Now().UnixNano(), Body: "hello", SeverityText: "info"}},
	}
}

func TestValidateBatch(t *testing.T) {
	batch := validBatch()
	if err := ValidateBatch(batch, DefaultLimits()); err != nil {
		t.Fatalf("valid batch rejected: %v", err)
	}
	batch.BatchID = "invalid"
	if err := ValidateBatch(batch, DefaultLimits()); err == nil {
		t.Fatal("invalid batch id accepted")
	}
}

func TestWriterRetriesAndDeduplicates(t *testing.T) {
	sink := &testSink{}
	sink.failures.Store(1)
	writer := NewWriter(WriterConfig{Workers: 1, MaxRetries: 1}, sink)
	defer writer.Close()

	if ack := writer.Submit(context.Background(), validBatch()); ack.ErrorCode != "" {
		t.Fatalf("write failed: %+v", ack)
	}
	if sink.calls.Load() != 2 {
		t.Fatalf("expected 2 sink calls, got %d", sink.calls.Load())
	}
	ack := writer.Submit(context.Background(), validBatch())
	if !ack.Duplicate || sink.calls.Load() != 2 {
		t.Fatalf("duplicate was written again: %+v calls=%d", ack, sink.calls.Load())
	}
}

func TestWriterConcurrentDuplicateWaitsForFirstWrite(t *testing.T) {
	sink := &testSink{started: make(chan struct{}), release: make(chan struct{})}
	sink.failures.Store(-1)
	writer := NewWriter(WriterConfig{Workers: 1}, sink)
	defer writer.Close()

	first := make(chan IngestAck, 1)
	go func() { first <- writer.Submit(context.Background(), validBatch()) }()
	<-sink.started
	second := make(chan IngestAck, 1)
	go func() { second <- writer.Submit(context.Background(), validBatch()) }()
	close(sink.release)
	if ack := <-first; ack.ErrorCode != "" {
		t.Fatalf("first write failed: %+v", ack)
	}
	if ack := <-second; ack.ErrorCode != "" || !ack.Duplicate {
		t.Fatalf("concurrent duplicate result invalid: %+v", ack)
	}
	if sink.calls.Load() != 1 {
		t.Fatalf("expected one sink call, got %d", sink.calls.Load())
	}
}

func TestGatewayAuthenticationRateLimitAndForwarding(t *testing.T) {
	sink := &testSink{}
	sink.failures.Store(-1)
	writer := NewWriter(WriterConfig{Workers: 1}, sink)
	defer writer.Close()
	writerServer := httptest.NewServer(WriterHTTPHandler(writer, "writer-token", 0))
	defer writerServer.Close()

	gateway := NewGateway(GatewayConfig{
		WriterURL: writerServer.URL, WriterToken: "writer-token", AgentTokens: []string{"agent-token"},
		RatePerSecond: 1, BurstRecords: 1,
	})
	if ack := gateway.Submit(context.Background(), "bad-token", validBatch()); ack.ErrorCode != "UNAUTHORIZED" {
		t.Fatalf("unexpected auth ack: %+v", ack)
	}
	if ack := gateway.Submit(context.Background(), "agent-token", validBatch()); ack.ErrorCode != "" {
		t.Fatalf("forward failed: %+v", ack)
	}
	batch := validBatch()
	batch.BatchID = "fdaf35fa-e311-418b-9629-e3d8c1f11702"
	if ack := gateway.Submit(context.Background(), "agent-token", batch); ack.ErrorCode != "RATE_LIMITED" {
		t.Fatalf("expected rate limit, got %+v", ack)
	}
}

func TestGatewayHTTPHandler(t *testing.T) {
	writerServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if bearerToken(request) != "writer-token" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		var batch LogBatch
		_ = json.NewDecoder(request.Body).Decode(&batch)
		writeJSON(writer, http.StatusOK, successAck(batch, false))
	}))
	defer writerServer.Close()
	gateway := NewGateway(GatewayConfig{WriterURL: writerServer.URL, WriterToken: "writer-token", AgentTokens: []string{"agent-token"}})
	server := httptest.NewServer(gateway.HTTPHandler())
	defer server.Close()

	raw, _ := json.Marshal(validBatch())
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/logs/batches", bytes.NewReader(raw))
	request.Header.Set("Authorization", "Bearer agent-token")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d", response.StatusCode)
	}
}

func TestGatewayAppliesGlobalRateLimitAcrossAgents(t *testing.T) {
	sink := &testSink{}
	sink.failures.Store(-1)
	writer := NewWriter(WriterConfig{Workers: 1}, sink)
	defer writer.Close()
	writerServer := httptest.NewServer(WriterHTTPHandler(writer, "writer-token", 0))
	defer writerServer.Close()

	gateway := NewGateway(GatewayConfig{
		WriterURL: writerServer.URL, WriterToken: "writer-token", AgentTokens: []string{"agent-token"},
		RatePerSecond: 1000, BurstRecords: 1000, GlobalRatePerSecond: 1, GlobalBurstRecords: 1,
	})
	defer gateway.Close()
	first := validBatch()
	first.AgentID = "agent-a"
	if ack := gateway.Submit(context.Background(), "agent-token", first); ack.ErrorCode != "" {
		t.Fatalf("first submit failed: %+v", ack)
	}
	second := validBatch()
	second.BatchID = "fdaf35fa-e311-418b-9629-e3d8c1f11702"
	second.AgentID = "agent-b"
	if ack := gateway.Submit(context.Background(), "agent-token", second); ack.ErrorCode != "RATE_LIMITED" {
		t.Fatalf("global rate limit was not applied: %+v", ack)
	}
}

type blockingPublisher struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (publisher *blockingPublisher) Publish(ctx context.Context, batch LogBatch) (IngestAck, error) {
	publisher.once.Do(func() { close(publisher.started) })
	select {
	case <-ctx.Done():
		return IngestAck{}, ctx.Err()
	case <-publisher.release:
		return successAck(batch, false), nil
	}
}

func (publisher *blockingPublisher) Close() {}

func TestGatewayRejectsRequestsAboveInflightLimit(t *testing.T) {
	publisher := &blockingPublisher{started: make(chan struct{}), release: make(chan struct{})}
	gateway := NewGateway(GatewayConfig{
		QueueMode: "kafka", Publisher: publisher, AgentTokens: []string{"agent-token"}, MaxInflight: 1,
		RatePerSecond: 1000, BurstRecords: 1000, GlobalRatePerSecond: 1000, GlobalBurstRecords: 1000,
	})
	defer gateway.Close()
	firstResult := make(chan IngestAck, 1)
	go func() { firstResult <- gateway.Submit(context.Background(), "agent-token", validBatch()) }()
	<-publisher.started

	second := validBatch()
	second.BatchID = "fdaf35fa-e311-418b-9629-e3d8c1f11702"
	if ack := gateway.Submit(context.Background(), "agent-token", second); ack.ErrorCode != "GATEWAY_OVERLOADED" {
		t.Fatalf("inflight limit was not applied: %+v", ack)
	}
	close(publisher.release)
	if ack := <-firstResult; ack.ErrorCode != "" {
		t.Fatalf("first request failed: %+v", ack)
	}
}

func TestClickHouseSinkUsesBatchDeduplicationToken(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		query = string(raw)
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sink := NewClickHouseSink(logmodel.StorageCluster{
		Endpoints: server.URL, DatabaseName: "opshub_logs", Timeout: 5,
	}, "")
	if err := sink.WriteBatch(context.Background(), validBatch()); err != nil {
		t.Fatalf("WriteBatch failed: %v", err)
	}
	if !strings.Contains(query, "insert_deduplication_token='"+testBatchID+"'") {
		t.Fatalf("ClickHouse insert has no batch deduplication token: %s", query)
	}
}
