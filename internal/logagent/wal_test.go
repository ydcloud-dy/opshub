package logagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ydcloud-dy/opshub/internal/loggingest"
)

func testEvent(body string) Event {
	now := time.Now()
	return Event{SourceID: "app", FilePath: "/var/log/app.log", Service: "api", Timestamp: now, ObservedAt: now, Body: body, Level: "INFO", Attributes: map[string]string{}}
}

func TestWALRecoversActiveSegmentAfterRestart(t *testing.T) {
	directory := t.TempDir()
	wal, err := OpenWAL(directory, 1024*1024, 10, &Metrics{})
	if err != nil {
		t.Fatal(err)
	}
	if err := wal.Append(testEvent("persist me")); err != nil {
		t.Fatal(err)
	}
	if err := wal.Close(false); err != nil {
		t.Fatal(err)
	}
	recovered, err := OpenWAL(directory, 1024*1024, 10, &Metrics{})
	if err != nil {
		t.Fatal(err)
	}
	segments, err := recovered.ReadySegments()
	if err != nil || len(segments) != 1 {
		t.Fatalf("segments=%v err=%v", segments, err)
	}
	events, err := recovered.ReadSegment(segments[0])
	if err != nil || len(events) != 1 || events[0].Body != "persist me" {
		t.Fatalf("events=%#v err=%v", events, err)
	}
}

func TestSenderKeepsSegmentUntilGatewayAck(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var batch loggingest.LogBatch
		_ = json.NewDecoder(request.Body).Decode(&batch)
		if attempts.Add(1) == 1 {
			writer.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(writer).Encode(loggingest.IngestAck{BatchID: batch.BatchID, ErrorCode: "WRITER_UNAVAILABLE", ErrorMessage: "offline"})
			return
		}
		_ = json.NewEncoder(writer).Encode(loggingest.IngestAck{BatchID: batch.BatchID, AcceptedRecords: len(batch.Records), AcceptedSequence: batch.SequenceEnd})
	}))
	defer server.Close()

	directory := t.TempDir()
	metrics := &Metrics{}
	wal, err := OpenWAL(directory, 1024*1024, 1, metrics)
	if err != nil {
		t.Fatal(err)
	}
	if err := wal.Append(testEvent("retry me")); err != nil {
		t.Fatal(err)
	}
	sender := NewSender(server.URL, "token", AgentIdentity{AgentID: "agent-1", AssetType: "host", AssetID: 1, HostID: 1}, metrics)
	if err := sender.ProcessReady(context.Background(), wal); err == nil {
		t.Fatal("first send unexpectedly succeeded")
	}
	segments, _ := wal.ReadySegments()
	if len(segments) != 1 {
		t.Fatalf("segment removed before ACK: %v", segments)
	}
	if err := sender.ProcessReady(context.Background(), wal); err != nil {
		t.Fatal(err)
	}
	segments, _ = wal.ReadySegments()
	if len(segments) != 0 {
		t.Fatalf("segment remains after ACK: %v", segments)
	}
	if metrics.Snapshot().OutputRecords != 1 || metrics.Snapshot().RetryTotal != 1 {
		t.Fatalf("unexpected metrics: %#v", metrics.Snapshot())
	}
}

func TestWALRejectsWritesAtCapacity(t *testing.T) {
	directory := t.TempDir()
	wal, err := OpenWAL(directory, 32, 10, &Metrics{})
	if err != nil {
		t.Fatal(err)
	}
	if err := wal.Append(testEvent("this record is larger than the WAL")); err != ErrWALFull {
		t.Fatalf("Append error = %v", err)
	}
	entries, _ := os.ReadDir(filepath.Clean(directory))
	if len(entries) != 0 {
		t.Fatalf("unexpected WAL files: %v", entries)
	}
}

func TestFileToGatewayPipeline(t *testing.T) {
	var received loggingest.LogBatch
	writer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer writer-token" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(response).Encode(loggingest.IngestAck{
			BatchID: received.BatchID, AcceptedRecords: len(received.Records), AcceptedSequence: received.SequenceEnd,
		})
	}))
	defer writer.Close()
	gateway := loggingest.NewGateway(loggingest.GatewayConfig{
		WriterURL: writer.URL, WriterToken: "writer-token", AgentTokens: []string{"agent-token"},
	})
	gatewayServer := httptest.NewServer(gateway.HTTPHandler())
	defer gatewayServer.Close()

	directory := t.TempDir()
	logPath := filepath.Join(directory, "application.log")
	if err := os.WriteFile(logPath, []byte("2026-07-14 INFO pipeline ready\n"), 0600); err != nil {
		t.Fatal(err)
	}
	metrics := &Metrics{}
	wal, err := OpenWAL(filepath.Join(directory, "wal"), 1024*1024, 1, metrics)
	if err != nil {
		t.Fatal(err)
	}
	tailer := newTestTailer(t, logPath, filepath.Join(directory, "state"), wal)
	if err := tailer.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	sender := NewSender(gatewayServer.URL, "agent-token", AgentIdentity{
		AgentID: "agent-1", AssetType: "host", AssetID: 7, HostID: 7,
	}, metrics)
	if err := sender.ProcessReady(context.Background(), wal); err != nil {
		t.Fatal(err)
	}
	if received.AgentID != "agent-1" || received.HostID != 7 || len(received.Records) != 1 {
		t.Fatalf("unexpected batch: %#v", received)
	}
	if received.Records[0].Body != "2026-07-14 INFO pipeline ready" || received.Records[0].SeverityText != "INFO" {
		t.Fatalf("unexpected record: %#v", received.Records[0])
	}
}
