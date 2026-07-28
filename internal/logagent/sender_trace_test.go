package logagent

import (
	"testing"
	"time"
)

func TestBuildBatchPreservesTraceContext(t *testing.T) {
	sender := NewSender("http://gateway", "token", AgentIdentity{AgentID: "agent-1", AssetType: "host", AssetID: 1}, nil)
	now := time.Now()
	batch := sender.buildBatch("segment-1", 0, []Event{{
		Sequence: 1, SourceID: "source-1", Timestamp: now, ObservedAt: now,
		Body: "request completed", Level: "INFO",
		TraceID: "84e63d5e6087ee3536774a6d1a845d8d", SpanID: "fb22686e58e091dd",
	}})
	if len(batch.Records) != 1 {
		t.Fatalf("unexpected record count: %d", len(batch.Records))
	}
	if batch.Records[0].TraceID != "84e63d5e6087ee3536774a6d1a845d8d" || batch.Records[0].SpanID != "fb22686e58e091dd" {
		t.Fatalf("trace context was not preserved: %#v", batch.Records[0])
	}
}
