package logagent

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	inputRecords   atomic.Uint64
	outputRecords  atomic.Uint64
	droppedRecords atomic.Uint64
	retryTotal     atomic.Uint64
	parseErrors    atomic.Uint64
	truncated      atomic.Uint64
	walBytes       atomic.Int64
	configVersion  atomic.Uint64
	lastMu         sync.RWMutex
	lastSuccess    time.Time
	lastError      string
}

type MetricsSnapshot struct {
	InputRecords   uint64    `json:"inputRecords"`
	OutputRecords  uint64    `json:"outputRecords"`
	DroppedRecords uint64    `json:"droppedRecords"`
	RetryTotal     uint64    `json:"retryTotal"`
	ParseErrors    uint64    `json:"parseErrors"`
	Truncated      uint64    `json:"truncated"`
	WALBytes       int64     `json:"walBytes"`
	ConfigVersion  uint64    `json:"configVersion"`
	LastSuccess    time.Time `json:"lastSuccess"`
	LastError      string    `json:"lastError"`
}

func (metrics *Metrics) Snapshot() MetricsSnapshot {
	metrics.lastMu.RLock()
	defer metrics.lastMu.RUnlock()
	return MetricsSnapshot{
		InputRecords: metrics.inputRecords.Load(), OutputRecords: metrics.outputRecords.Load(),
		DroppedRecords: metrics.droppedRecords.Load(), RetryTotal: metrics.retryTotal.Load(),
		ParseErrors: metrics.parseErrors.Load(), Truncated: metrics.truncated.Load(),
		WALBytes: metrics.walBytes.Load(), ConfigVersion: metrics.configVersion.Load(),
		LastSuccess: metrics.lastSuccess, LastError: metrics.lastError,
	}
}

func (metrics *Metrics) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/metrics", func(writer http.ResponseWriter, _ *http.Request) {
		snapshot := metrics.Snapshot()
		writer.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprintf(writer, "opshub_log_agent_input_records_total %d\n", snapshot.InputRecords)
		_, _ = fmt.Fprintf(writer, "opshub_log_agent_output_records_total %d\n", snapshot.OutputRecords)
		_, _ = fmt.Fprintf(writer, "opshub_log_agent_dropped_records_total %d\n", snapshot.DroppedRecords)
		_, _ = fmt.Fprintf(writer, "opshub_log_agent_retry_total %d\n", snapshot.RetryTotal)
		_, _ = fmt.Fprintf(writer, "opshub_log_agent_parse_errors_total %d\n", snapshot.ParseErrors)
		_, _ = fmt.Fprintf(writer, "opshub_log_agent_truncated_records_total %d\n", snapshot.Truncated)
		_, _ = fmt.Fprintf(writer, "opshub_log_agent_wal_bytes %d\n", snapshot.WALBytes)
		_, _ = fmt.Fprintf(writer, "opshub_log_agent_config_version %d\n", snapshot.ConfigVersion)
		if !snapshot.LastSuccess.IsZero() {
			_, _ = fmt.Fprintf(writer, "opshub_log_agent_last_success_timestamp %d\n", snapshot.LastSuccess.Unix())
		}
	})
	return mux
}

func (metrics *Metrics) recordSuccess(records int) {
	metrics.outputRecords.Add(uint64(records))
	metrics.lastMu.Lock()
	metrics.lastSuccess = time.Now()
	metrics.lastError = ""
	metrics.lastMu.Unlock()
}

func (metrics *Metrics) recordDropped(records int) {
	if records > 0 {
		metrics.droppedRecords.Add(uint64(records))
	}
}

func (metrics *Metrics) recordError(err error) {
	metrics.lastMu.Lock()
	metrics.lastError = err.Error()
	metrics.lastMu.Unlock()
}
