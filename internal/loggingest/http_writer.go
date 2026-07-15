package loggingest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func WriterHTTPHandler(writerService *Writer, internalToken string, maxBodyBytes int64) http.Handler {
	if maxBodyBytes <= 0 {
		maxBodyBytes = 4 * 1024 * 1024
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(writer http.ResponseWriter, _ *http.Request) {
		status := writerService.Status()
		httpStatus := http.StatusOK
		if status.Status != "healthy" {
			httpStatus = http.StatusServiceUnavailable
		}
		writeJSON(writer, httpStatus, map[string]string{"status": status.Status})
	})
	mux.HandleFunc("/status", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, writerService.Status())
	})
	mux.HandleFunc("/metrics", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; version=0.0.4")
		status := writerService.Status()
		_, _ = fmt.Fprintf(writer, "opshub_log_writer_accepted_batches_total %d\n", status.AcceptedBatches)
		_, _ = fmt.Fprintf(writer, "opshub_log_writer_accepted_records_total %d\n", status.AcceptedRecords)
		_, _ = fmt.Fprintf(writer, "opshub_log_writer_failed_batches_total %d\n", status.FailedBatches)
		_, _ = fmt.Fprintf(writer, "opshub_log_writer_queue_depth %d\n", status.QueueDepth)
		_, _ = fmt.Fprintf(writer, "opshub_log_writer_queue_lag %d\n", status.QueueLag)
		_, _ = fmt.Fprintf(writer, "opshub_log_writer_write_latency_ms %.3f\n", status.WriteLatencyMS)
		_, _ = fmt.Fprintf(writer, "opshub_log_writer_deadletter_batches_total %d\n", status.DeadletterBatches)
		_, _ = fmt.Fprintf(writer, "opshub_log_writer_queue_healthy %d\n", boolMetric(status.QueueHealthy))
	})
	mux.HandleFunc("/internal/v1/write", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if internalToken == "" || bearerToken(request) != internalToken {
			writeJSON(writer, http.StatusUnauthorized, errorAck("", "UNAUTHORIZED", "Writer Token 无效", 0))
			return
		}
		request.Body = http.MaxBytesReader(writer, request.Body, maxBodyBytes)
		var batch LogBatch
		if err := json.NewDecoder(request.Body).Decode(&batch); err != nil {
			writeJSON(writer, http.StatusBadRequest, errorAck(batch.BatchID, "INVALID_JSON", err.Error(), 0))
			return
		}
		if err := ValidateBatch(batch, DefaultLimits()); err != nil {
			writeJSON(writer, http.StatusBadRequest, errorAck(batch.BatchID, "INVALID_BATCH", err.Error(), 0))
			return
		}
		ack := writerService.Submit(request.Context(), batch)
		status := http.StatusOK
		if ack.ErrorCode != "" {
			status = http.StatusServiceUnavailable
			if strings.HasPrefix(ack.ErrorCode, "INVALID") {
				status = http.StatusBadRequest
			}
		}
		writeJSON(writer, status, ack)
	})
	return mux
}
