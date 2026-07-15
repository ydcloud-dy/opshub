package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/pkg/response"
	logsvc "github.com/ydcloud-dy/opshub/plugins/logcenter/service"
)

const tailBatchSize = 500

func (h *Handler) TailInternalLogs(c *gin.Context) {
	var req logsvc.InternalQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if !h.applyInternalPermissions(c, &req, "tail") {
		return
	}
	cluster, password, ok := h.internalStorage(c, req.StorageID)
	if !ok {
		return
	}
	if h.tailSlots != nil {
		select {
		case h.tailSlots <- struct{}{}:
			defer func() { <-h.tailSlots }()
		default:
			response.ErrorCode(c, http.StatusTooManyRequests, "实时 Tail 连接数已达到上限，请稍后重试")
			return
		}
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.ErrorCode(c, http.StatusInternalServerError, "当前 HTTP 服务不支持流式响应")
		return
	}
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Time{})
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	flusher.Flush()

	startedAt := time.Now().UTC()
	if !writeTailEvent(c, flusher, "ready", gin.H{"startedAt": startedAt.Format(time.RFC3339Nano)}) {
		return
	}
	pollInterval := time.Duration(positiveEnvInt("OPSHUB_LOG_TAIL_POLL_MILLISECONDS", 1000)) * time.Millisecond
	if pollInterval < 500*time.Millisecond {
		pollInterval = 500 * time.Millisecond
	}
	maxDuration := time.Duration(positiveEnvInt("OPSHUB_LOG_TAIL_MAX_MINUTES", 60)) * time.Minute
	deadline := time.NewTimer(maxDuration)
	defer deadline.Stop()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	req.Sort = "asc"
	req.Limit = tailBatchSize
	req.Cursor = ""
	req.SkipHistory = true
	tailStart := startedAt.Add(-2 * time.Second)
	consecutiveErrors := 0
	for {
		now := time.Now().UTC()
		req.Start = tailStart.Format(time.RFC3339Nano)
		req.End = now.Add(time.Second).Format(time.RFC3339Nano)
		result, err := h.internalQuery.Query(c.Request.Context(), cluster, password, req)
		if err != nil {
			consecutiveErrors++
			if !writeTailEvent(c, flusher, "error", gin.H{"message": err.Error(), "retrying": consecutiveErrors < 5}) {
				return
			}
			if consecutiveErrors >= 5 {
				return
			}
		} else {
			consecutiveErrors = 0
			if len(result.Items) > 0 {
				req.Cursor = result.NextCursor
				if !writeTailEvent(c, flusher, "logs", gin.H{
					"items": result.Items, "cursor": result.NextCursor, "receivedAt": now.Format(time.RFC3339Nano),
				}) {
					return
				}
			}
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-deadline.C:
			_ = writeTailEvent(c, flusher, "end", gin.H{"reason": "max_duration"})
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprintf(c.Writer, ": heartbeat %d\n\n", time.Now().Unix()); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
		}
	}
}

func writeTailEvent(c *gin.Context, flusher http.Flusher, event string, value interface{}) bool {
	raw, err := json.Marshal(value)
	if err != nil {
		return false
	}
	if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, raw); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

func positiveEnvInt(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
