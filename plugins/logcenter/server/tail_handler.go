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
	rbacsvc "github.com/ydcloud-dy/opshub/internal/service/rbac"
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
	maxDuration := time.Duration(positiveEnvInt("OPSHUB_LOG_TAIL_MAX_MINUTES", 1440)) * time.Minute
	deadlineDuration, deadlineReason := tailStreamDeadline(maxDuration, rbacsvc.GetTokenExpiresAt(c), time.Now())
	deadline := time.NewTimer(deadlineDuration)
	defer deadline.Stop()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	reauthorize := time.NewTicker(tailReauthorizationInterval())
	defer reauthorize.Stop()

	req.Sort = "asc"
	req.Limit = tailBatchSize
	req.SkipHistory = true
	tailStart := startedAt.Add(-2 * time.Second)
	// Reconnects carry the last cursor and timestamp so a short interruption does not create a gap.
	if strings.TrimSpace(req.Cursor) != "" {
		if requestedStart := parseTailTime(req.Start); !requestedStart.IsZero() {
			tailStart = requestedStart
		}
	}
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
			if deadlineReason == "token_expired" {
				_ = writeTailEvent(c, flusher, "error", gin.H{"message": "登录凭证已过期", "status": http.StatusUnauthorized, "retrying": false})
			}
			_ = writeTailEvent(c, flusher, "end", gin.H{"reason": deadlineReason})
			return
		case <-reauthorize.C:
			decision, failure := h.resolveInternalAction(c, "tail", req.StorageID)
			if failure != nil {
				_ = writeTailEvent(c, flusher, "error", gin.H{"message": failure.Message, "status": failure.Status, "retrying": false})
				_ = writeTailEvent(c, flusher, "end", gin.H{"reason": "authorization_changed"})
				return
			}
			if message := queryFieldAccessError(req, decision.DeniedFields); message != "" {
				_ = writeTailEvent(c, flusher, "error", gin.H{"message": message, "status": http.StatusForbidden, "retrying": false})
				_ = writeTailEvent(c, flusher, "end", gin.H{"reason": "authorization_changed"})
				return
			}
			applyTailAccessDecision(&req, decision)
		case <-heartbeat.C:
			if _, err := fmt.Fprintf(c.Writer, ": heartbeat %d\n\n", time.Now().Unix()); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
		}
	}
}

func applyTailAccessDecision(req *logsvc.InternalQueryRequest, decision logAccessDecision) {
	if decision.IsAdmin {
		req.AllowedPolicyIDs = nil
		req.AllowedHostIDs = nil
		req.AllowedKubernetesScopes = nil
		req.DeniedFields = nil
		req.MaskFields = nil
		return
	}
	req.AllowedPolicyIDs = decision.AllowedPolicyIDs
	req.AllowedHostIDs = decision.AllowedHostIDs
	req.AllowedKubernetesScopes = decision.AllowedKubernetesScopes
	req.DeniedFields = decision.DeniedFields
	req.MaskFields = decision.MaskFields
}

func tailReauthorizationInterval() time.Duration {
	seconds := positiveEnvInt("OPSHUB_LOG_TAIL_REAUTH_SECONDS", 15)
	if seconds < 5 {
		seconds = 5
	}
	if seconds > 300 {
		seconds = 300
	}
	return time.Duration(seconds) * time.Second
}

func tailStreamDeadline(maxDuration time.Duration, expiresAt, now time.Time) (time.Duration, string) {
	if !expiresAt.IsZero() {
		remaining := expiresAt.Sub(now)
		if remaining <= 0 {
			return time.Millisecond, "token_expired"
		}
		if remaining < maxDuration {
			return remaining, "token_expired"
		}
	}
	return maxDuration, "max_duration"
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

func parseTailTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}
