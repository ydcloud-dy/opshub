package logagent

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type memorySink struct {
	mutex  sync.Mutex
	events []Event
}

func (sink *memorySink) Append(event Event) error {
	sink.mutex.Lock()
	defer sink.mutex.Unlock()
	sink.events = append(sink.events, event)
	return nil
}

func newTestTailer(t *testing.T, pattern, stateDir string, sink EventSink) *Tailer {
	t.Helper()
	config := Config{Enabled: true, GatewayURL: "http://gateway", GatewayToken: "token", StateDir: stateDir, Sources: []SourceConfig{{
		ID: "test", Paths: []string{pattern}, ReadFrom: "beginning", MaxLineBytes: 1024, Parser: ParserConfig{Type: "raw"},
	}}}
	store, err := NewCheckpointStore(filepath.Join(stateDir, "checkpoints.json"))
	if err != nil {
		t.Fatal(err)
	}
	tailer, err := NewTailer(config, store, sink, &Metrics{})
	if err != nil {
		t.Fatal(err)
	}
	return tailer
}

func TestTailerCheckpointPreventsDuplicateAfterRestart(t *testing.T) {
	directory := t.TempDir()
	logPath := filepath.Join(directory, "app.log")
	if err := os.WriteFile(logPath, []byte("first\nsecond\n"), 0600); err != nil {
		t.Fatal(err)
	}
	sink := &memorySink{}
	tailer := newTestTailer(t, logPath, directory, sink)
	if err := tailer.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	tailer.closeAll()
	restarted := newTestTailer(t, logPath, directory, sink)
	if err := restarted.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sink.events) != 2 {
		t.Fatalf("events = %d, want 2", len(sink.events))
	}
}

func TestTailerHandlesCopytruncate(t *testing.T) {
	directory := t.TempDir()
	logPath := filepath.Join(directory, "app.log")
	if err := os.WriteFile(logPath, []byte("before rotation\n"), 0600); err != nil {
		t.Fatal(err)
	}
	sink := &memorySink{}
	tailer := newTestTailer(t, logPath, directory, sink)
	if err := tailer.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("after rotation\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tailer.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sink.events) != 2 || sink.events[1].Body != "after rotation" {
		t.Fatalf("unexpected events: %#v", sink.events)
	}
}

func TestTailerDetectsCopytruncateAfterFileRegrows(t *testing.T) {
	directory := t.TempDir()
	logPath := filepath.Join(directory, "app.log")
	oldLine := strings.Repeat("A", 96)
	newLine := strings.Repeat("B", 128)
	if err := os.WriteFile(logPath, []byte(oldLine+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	sink := &memorySink{}
	tailer := newTestTailer(t, logPath, directory, sink)
	if err := tailer.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte(newLine+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tailer.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sink.events) != 2 || sink.events[1].Body != newLine {
		t.Fatalf("copytruncate was not detected: %#v", sink.events)
	}
}

func TestTailerHandlesRenameRotation(t *testing.T) {
	directory := t.TempDir()
	logPath := filepath.Join(directory, "app.log")
	if err := os.WriteFile(logPath, []byte("old file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	sink := &memorySink{}
	tailer := newTestTailer(t, filepath.Join(directory, "app.log*"), directory, sink)
	if err := tailer.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(logPath, logPath+".1"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("new file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tailer.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sink.events) != 2 || sink.events[1].Body != "new file" {
		t.Fatalf("unexpected events: %#v", sink.events)
	}
}

func TestReadBoundedLineTruncatesWithoutLosingFollowingLine(t *testing.T) {
	reader := bufio.NewReaderSize(strings.NewReader("123456789\nnext\n"), 4)
	line, _, truncated, err := readBoundedLine(reader, 5)
	if err != nil || line != "12345" || !truncated {
		t.Fatalf("first line = %q truncated=%v err=%v", line, truncated, err)
	}
	next, _, _, err := readBoundedLine(reader, 5)
	if err != nil || next != "next" {
		t.Fatalf("next line = %q err=%v", next, err)
	}
}
