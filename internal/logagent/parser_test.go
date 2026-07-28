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

func TestRawParserExtractsTraceContext(t *testing.T) {
	parser, err := NewLineParser(ParserConfig{Type: "raw"})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parser.Parse("2026-07-23 10:35:15.023 INFO trace_id=84e63d5e6087ee3536774a6d1a845d8d span_id=fb22686e58e091dd request completed", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.TraceID != "84e63d5e6087ee3536774a6d1a845d8d" || parsed.SpanID != "fb22686e58e091dd" {
		t.Fatalf("unexpected trace context: trace=%q span=%q", parsed.TraceID, parsed.SpanID)
	}
	if parsed.Level != "INFO" {
		t.Fatalf("trace_id must not override the INFO level: %q", parsed.Level)
	}
}

func TestRawParserDetectsStandaloneLogLevels(t *testing.T) {
	parser, err := NewLineParser(ParserConfig{Type: "raw"})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		line  string
		level string
	}{
		{"2026-07-23 13:29:12.214 INFO 1 --- trace_id=05196f17435f09512c016165ae4753d9 request completed", "INFO"},
		{"2026-07-23 13:29:12 [WARNING] queue is nearly full", "WARN"},
		{"2026-07-23 13:29:12 [TRACE] request details", "TRACE"},
		{"trace_id=05196f17435f09512c016165ae4753d9 span_id=8a6a75c8d7699ef9 request completed", "INFO"},
	}
	for _, test := range tests {
		parsed, parseErr := parser.Parse(test.line, time.Now())
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		if parsed.Level != test.level {
			t.Fatalf("line %q: expected %s, got %s", test.line, test.level, parsed.Level)
		}
	}
}

func TestJSONParserExtractsTraceContext(t *testing.T) {
	parser, err := NewLineParser(ParserConfig{Type: "json"})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parser.Parse(`{"message":"request completed","trace_id":"84e63d5e6087ee3536774a6d1a845d8d","spanId":"fb22686e58e091dd"}`, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.TraceID != "84e63d5e6087ee3536774a6d1a845d8d" || parsed.SpanID != "fb22686e58e091dd" {
		t.Fatalf("unexpected trace context: trace=%q span=%q", parsed.TraceID, parsed.SpanID)
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

func TestRegexParserExtractsNamedTraceContext(t *testing.T) {
	parser, err := NewLineParser(ParserConfig{
		Type: "regex", Pattern: `^(?P<timestamp>\S+) (?P<level>\S+) trace_id=(?P<trace_id>\S+) span_id=(?P<span_id>\S+) (?P<message>.*)$`,
	})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parser.Parse("2026-07-23T02:35:15Z INFO trace_id=84e63d5e6087ee3536774a6d1a845d8d span_id=fb22686e58e091dd request completed", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.TraceID != "84e63d5e6087ee3536774a6d1a845d8d" || parsed.SpanID != "fb22686e58e091dd" {
		t.Fatalf("unexpected trace context: trace=%q span=%q", parsed.TraceID, parsed.SpanID)
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

func TestGoMultilineAssembler(t *testing.T) {
	assembler, err := NewMultilineAssembler(MultilineConfig{Enabled: true, Preset: "go"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	assembler.Add("panic: runtime error: index out of range", now)
	assembler.Add("goroutine 1 [running]:", now)
	assembler.Add("main.main()", now)
	assembler.Add("\t/app/main.go:10 +0x20", now)
	values := assembler.Add("2026-07-14 INFO recovered", now)
	want := "panic: runtime error: index out of range\ngoroutine 1 [running]:\nmain.main()\n\t/app/main.go:10 +0x20"
	if len(values) != 1 || values[0] != want {
		t.Fatalf("unexpected multiline result: %#v", values)
	}
}

func TestPythonMultilineAssembler(t *testing.T) {
	assembler, err := NewMultilineAssembler(MultilineConfig{Enabled: true, Preset: "python"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	assembler.Add("Traceback (most recent call last):", now)
	assembler.Add(`  File "/app/main.py", line 10, in <module>`, now)
	assembler.Add("    run()", now)
	assembler.Add("ValueError: invalid input", now)
	values := assembler.Add("2026-07-14 INFO recovered", now)
	want := "Traceback (most recent call last):\n  File \"/app/main.py\", line 10, in <module>\n    run()\nValueError: invalid input"
	if len(values) != 1 || values[0] != want {
		t.Fatalf("unexpected multiline result: %#v", values)
	}
}

func TestCustomMultilineAssembler(t *testing.T) {
	assembler, err := NewMultilineAssembler(MultilineConfig{Enabled: true, Preset: "custom", StartPattern: `^\d{4}-\d{2}-\d{2}`})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	assembler.Add("2026-07-14 ERROR request failed", now)
	assembler.Add("custom continuation", now)
	values := assembler.Add("2026-07-14 INFO recovered", now)
	if len(values) != 1 || values[0] != "2026-07-14 ERROR request failed\ncustom continuation" {
		t.Fatalf("unexpected multiline result: %#v", values)
	}
}
