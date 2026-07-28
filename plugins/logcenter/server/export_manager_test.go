package server

import (
	"path/filepath"
	"testing"
	"time"

	logsvc "github.com/ydcloud-dy/opshub/plugins/logcenter/service"
)

func TestLogExportPayloadPreservesPermissionSnapshot(t *testing.T) {
	request := logsvc.InternalQueryRequest{
		Start: "2026-07-22T00:00:00Z", End: "2026-07-22T01:00:00Z", Query: "error",
		AllowedPolicyIDs: []uint64{11, 13},
		AllowedHostIDs:   []uint64{7, 9}, AllowedKubernetesScopes: map[uint64][]string{3: {"prod"}},
		DeniedFields: []string{"secret"}, MaskFields: []string{"token"}, DenyAll: true,
		RequiredFilters: []logsvc.InternalQueryFilter{{Field: "environment", Operator: "eq", Value: "prod"}},
	}
	raw, err := encodeLogExportRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := decodeLogExportPayload(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload.AllowedPolicyIDs) != 2 || payload.AllowedPolicyIDs[0] != 11 || len(payload.AllowedHostIDs) != 2 || payload.AllowedKubernetesScopes[3][0] != "prod" || !payload.DenyAll {
		t.Fatalf("permission snapshot was not preserved: %#v", payload)
	}
	if len(payload.RequiredFilters) != 1 || payload.RequiredFilters[0].Field != "environment" {
		t.Fatalf("required filters were not preserved: %#v", payload.RequiredFilters)
	}
}

func TestLogExportRejectsLegacyPayloadWithoutPermissionSnapshot(t *testing.T) {
	if _, err := decodeLogExportPayload(`{"start":"2026-07-22T00:00:00Z","end":"2026-07-22T01:00:00Z"}`); err == nil {
		t.Fatal("legacy export payload unexpectedly accepted")
	}
}

func TestLogExportArtifactPathStaysInsideDirectory(t *testing.T) {
	directory := t.TempDir()
	manager := &logExportManager{directory: directory}
	path, err := manager.secureArtifactPath("logs.ndjson")
	if err != nil || path != filepath.Join(directory, "logs.ndjson") {
		t.Fatalf("path=%q err=%v", path, err)
	}
	if _, err := manager.secureArtifactPath("../outside.ndjson"); err == nil {
		t.Fatal("path traversal unexpectedly accepted")
	}
}

func TestLogExportRetryDelayIsBounded(t *testing.T) {
	if exportRetryDelay(1) != time.Second || exportRetryDelay(99) != 128*time.Second {
		t.Fatalf("unexpected retry delays: %s %s", exportRetryDelay(1), exportRetryDelay(99))
	}
}

func TestSanitizeCSVCellPreventsSpreadsheetFormulaInjection(t *testing.T) {
	for _, value := range []string{"=1+1", "+cmd|' /C calc'!A0", "-2+3", "@SUM(A1:A2)", "  =HYPERLINK(\"https://example.com\")"} {
		if sanitized := sanitizeCSVCell(value); sanitized != "'"+value {
			t.Fatalf("value %q was sanitized as %q", value, sanitized)
		}
	}
	for _, value := range []string{"normal", "2026-07-27T10:00:00Z", "", "{\"level\":\"INFO\"}"} {
		if sanitized := sanitizeCSVCell(value); sanitized != value {
			t.Fatalf("safe value %q changed to %q", value, sanitized)
		}
	}
}
