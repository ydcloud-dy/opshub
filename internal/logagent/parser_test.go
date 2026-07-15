package logagent

import (
	"testing"
	"time"
)

func TestJSONParserExtractsStandardFields(t *testing.T) {
	parser, err := NewLineParser(ParserConfig{Type: "json"})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parser.Parse(`{"timestamp":"2026-07-14T01:02:03Z","level":"error","message":"database unavailable","service":"api"}`, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Body != "database unavailable" || parsed.Level != "ERROR" || parsed.Attributes["service"] != "api" {
		t.Fatalf("unexpected parsed record: %#v", parsed)
	}
	if parsed.Timestamp.UTC().Format(time.RFC3339) != "2026-07-14T01:02:03Z" {
		t.Fatalf("unexpected timestamp: %s", parsed.Timestamp)
	}
}

func TestRegexParserUsesNamedGroups(t *testing.T) {
	parser, err := NewLineParser(ParserConfig{
		Type: "regex", Pattern: `^(?P<timestamp>\S+) (?P<level>\S+) (?P<message>.*)$`,
	})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parser.Parse("2026-07-14T01:02:03Z WARN queue is full", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Body != "queue is full" || parsed.Level != "WARN" {
		t.Fatalf("unexpected parsed record: %#v", parsed)
	}
}

func TestJavaMultilineAssembler(t *testing.T) {
	assembler, err := NewMultilineAssembler(MultilineConfig{Enabled: true, Preset: "java"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if values := assembler.Add("2026-07-14 ERROR request failed", now); len(values) != 0 {
		t.Fatalf("unexpected flush: %#v", values)
	}
	assembler.Add("    at app.Service.call(Service.java:10)", now)
	values := assembler.Add("2026-07-14 INFO recovered", now)
	if len(values) != 1 || values[0] != "2026-07-14 ERROR request failed\n    at app.Service.call(Service.java:10)" {
		t.Fatalf("unexpected multiline result: %#v", values)
	}
}
