package logagent

import (
	"testing"
	"time"
)

func TestCRIAssemblerMergesPartialRecords(t *testing.T) {
	assembler := NewCRIAssembler(1024)
	first := assembler.Add("2026-07-14T01:02:03.123456789Z stdout P hello ", time.Now())
	if len(first) != 0 {
		t.Fatalf("partial record flushed early: %#v", first)
	}
	result := assembler.Add("2026-07-14T01:02:03.223456789Z stdout F world", time.Now())
	if len(result) != 1 {
		t.Fatalf("records = %d", len(result))
	}
	if result[0].Body != "hello world" || result[0].Stream != "stdout" {
		t.Fatalf("unexpected record: %#v", result[0])
	}
	if result[0].Attributes["log.cri_partial_merged"] != "true" {
		t.Fatalf("partial marker missing: %#v", result[0].Attributes)
	}
}

func TestCRIAssemblerAcceptsDockerJSON(t *testing.T) {
	assembler := NewCRIAssembler(1024)
	result := assembler.Add(`{"log":"request failed\n","stream":"stderr","time":"2026-07-14T01:02:03.123456789Z"}`, time.Now())
	if len(result) != 1 || result[0].Body != "request failed" || result[0].Stream != "stderr" {
		t.Fatalf("unexpected docker record: %#v", result)
	}
	if result[0].Timestamp.UTC().Format(time.RFC3339Nano) != "2026-07-14T01:02:03.123456789Z" {
		t.Fatalf("timestamp = %s", result[0].Timestamp)
	}
}

func TestSenderBuildsKubernetesBatchMetadata(t *testing.T) {
	sender := NewSender("http://gateway", "token", AgentIdentity{
		AgentID: "k8s:7:worker-01", AssetType: "kubernetes", AssetID: 7, ClusterID: 7, NodeName: "worker-01",
	}, nil)
	event := Event{
		Sequence: 1, SourceID: "policy-1", PolicyID: 1, PolicyVersion: 2,
		FilePath: "/var/log/containers/api_default_api-abc.log", Environment: "production", Service: "api",
		Stream: "stderr", Namespace: "default", WorkloadKind: "Deployment", WorkloadName: "api",
		PodName: "api-abc", PodUID: "pod-uid", ContainerName: "api", ContainerImage: "api:v1", NodeName: "worker-01",
		Timestamp: time.Now(), ObservedAt: time.Now(), Body: "failed", Level: "ERROR",
	}
	batch := sender.buildBatch("segment", 0, []Event{event})
	if batch.SourceType != "kubernetes" || batch.ClusterID != 7 || batch.Namespace != "default" {
		t.Fatalf("unexpected batch identity: %#v", batch)
	}
	if batch.WorkloadKind != "Deployment" || batch.PodUID != "pod-uid" || batch.ContainerName != "api" {
		t.Fatalf("unexpected batch metadata: %#v", batch)
	}
}
