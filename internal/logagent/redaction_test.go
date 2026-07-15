package logagent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultRedactorRemovesSensitiveValues(t *testing.T) {
	redactor, err := NewRedactor(DefaultRedactionConfig())
	if err != nil {
		t.Fatal(err)
	}
	event := Event{
		Body:               `{"user":"demo","password":"plain-password","nested":{"token":"plain-token"}}`,
		Attributes:         map[string]string{"authorization": "Bearer plain-auth", "request.id": "123"},
		ResourceAttributes: map[string]string{"cookie": "session=plain-cookie"},
	}
	redactor.Apply(&event)
	serialized := event.Body + event.Attributes["authorization"] + event.ResourceAttributes["cookie"]
	for _, secret := range []string{"plain-password", "plain-token", "plain-auth", "plain-cookie"} {
		if strings.Contains(serialized, secret) {
			t.Fatalf("secret %q survived redaction: %s", secret, serialized)
		}
	}
	if !strings.Contains(event.Body, redactedValue) || event.Attributes["authorization"] != redactedValue {
		t.Fatalf("redaction result is incomplete: %#v", event)
	}
}

func TestRedactorSupportsHashDropAndRegex(t *testing.T) {
	config := RedactionConfig{
		Configured: true, Enabled: true,
		Rules: []RedactionRule{
			{Target: "field", Field: "user.phone", Action: "hash"},
			{Target: "field", Field: "debug", Action: "drop_field"},
			{Target: "regex", Pattern: `AK[A-Z0-9]{8}`, Action: "replace", Replacement: "AK***"},
		},
	}
	redactor, err := NewRedactor(config)
	if err != nil {
		t.Fatal(err)
	}
	event := Event{Body: "credential=AK12345678", Attributes: map[string]string{"user.phone": "13800138000", "debug": "secret"}}
	redactor.Apply(&event)
	if event.Body != "credential=AK***" || !strings.HasPrefix(event.Attributes["user.phone"], "sha256:") {
		t.Fatalf("custom rules were not applied: %#v", event)
	}
	if _, exists := event.Attributes["debug"]; exists {
		t.Fatalf("drop_field did not remove the field: %#v", event.Attributes)
	}
}

func TestTailerRedactsBeforeWAL(t *testing.T) {
	directory := t.TempDir()
	logPath := filepath.Join(directory, "app.log")
	if err := os.WriteFile(logPath, []byte("password=wal-secret token=token-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	metrics := &Metrics{}
	wal, err := OpenWAL(filepath.Join(directory, "wal"), 1024*1024, 1, metrics)
	if err != nil {
		t.Fatal(err)
	}
	checkpoints, err := NewCheckpointStore(filepath.Join(directory, "checkpoints.json"))
	if err != nil {
		t.Fatal(err)
	}
	config := Config{Enabled: true, GatewayURL: "http://gateway", GatewayToken: "token", StateDir: directory, Sources: []SourceConfig{{
		ID: "secure", Paths: []string{logPath}, ReadFrom: "beginning", MaxLineBytes: 1024,
		Parser: ParserConfig{Type: "raw"},
	}}}
	tailer, err := NewTailer(config, checkpoints, wal, metrics)
	if err != nil {
		t.Fatal(err)
	}
	if err := tailer.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := wal.Close(true); err != nil {
		t.Fatal(err)
	}
	segments, err := wal.ReadySegments()
	if err != nil || len(segments) != 1 {
		t.Fatalf("WAL segments = %#v, err = %v", segments, err)
	}
	raw, err := os.ReadFile(segments[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "wal-secret") || strings.Contains(string(raw), "token-secret") {
		t.Fatalf("plain secret reached WAL: %s", raw)
	}
}

func TestRetentionConfigUsesLevelOverride(t *testing.T) {
	config := RetentionConfig{DefaultDays: 7, LevelDays: map[string]int{"error": 30}}
	config.normalize()
	if config.DaysForLevel("ERROR") != 30 || config.DaysForLevel("INFO") != 7 {
		t.Fatalf("unexpected retention config: %#v", config)
	}
}
