package logagent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ydcloud-dy/opshub/internal/loggingest"
)

type Sender struct {
	gatewayURL string
	token      string
	identity   AgentIdentity
	client     *http.Client
	metrics    *Metrics
}

type batchError struct {
	code    string
	message string
}

func (err *batchError) Error() string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("Gateway 拒绝批次: %s %s", err.code, err.message)
}

func isPermanentBatchError(err error) bool {
	var batchErr *batchError
	return errors.As(err, &batchErr) && batchErr.code == "INVALID_BATCH"
}

func NewSender(gatewayURL, token string, identity AgentIdentity, metrics *Metrics) *Sender {
	if metrics == nil {
		metrics = &Metrics{}
	}
	return &Sender{
		gatewayURL: strings.TrimRight(gatewayURL, "/"), token: token, identity: identity,
		client: &http.Client{Timeout: 30 * time.Second}, metrics: metrics,
	}
}

func (sender *Sender) ProcessReady(ctx context.Context, wal *WAL) error {
	segments, err := wal.ReadySegments()
	if err != nil {
		return err
	}
	for _, segment := range segments {
		events, err := wal.ReadSegment(segment)
		if err != nil {
			return err
		}
		events, dropped := filterUnsendableEvents(events, time.Now())
		sender.metrics.recordDropped(dropped)
		if len(events) == 0 {
			if err := wal.DeleteSegment(segment); err != nil {
				return err
			}
			continue
		}
		groups := groupEvents(events)
		for index, group := range groups {
			batch := sender.buildBatch(filepath.Base(segment), index, group)
			if err := sender.sendBatch(ctx, batch); err != nil {
				if isPermanentBatchError(err) {
					sender.metrics.recordDropped(len(group))
					sender.metrics.recordError(err)
					continue
				}
				sender.metrics.retryTotal.Add(1)
				sender.metrics.recordError(err)
				return err
			}
			sender.metrics.recordSuccess(len(group))
		}
		if err := wal.DeleteSegment(segment); err != nil {
			return err
		}
	}
	return nil
}

func filterUnsendableEvents(events []Event, now time.Time) ([]Event, int) {
	limits := loggingest.DefaultLimits()
	kept := make([]Event, 0, len(events))
	dropped := 0
	for _, event := range events {
		if event.Timestamp.IsZero() || len(event.Body) > limits.MaxRecordBytes {
			dropped++
			continue
		}
		if limits.MaxFutureSkew > 0 && event.Timestamp.After(now.Add(limits.MaxFutureSkew)) {
			dropped++
			continue
		}
		if limits.MaxPastAge > 0 && event.Timestamp.Before(now.Add(-limits.MaxPastAge)) {
			dropped++
			continue
		}
		if event.RetentionDays > 0 && event.Timestamp.Before(now.Add(-time.Duration(event.RetentionDays)*24*time.Hour)) {
			dropped++
			continue
		}
		kept = append(kept, event)
	}
	return kept, dropped
}

func (sender *Sender) buildBatch(segment string, groupIndex int, events []Event) loggingest.LogBatch {
	first := events[0]
	batch := loggingest.LogBatch{
		BatchID: deterministicBatchID(segment, groupIndex, first.SourceID), AgentID: sender.identity.AgentID,
		PolicyID: first.PolicyID, PolicyVersion: first.PolicyVersion,
		SequenceStart: events[0].Sequence, SequenceEnd: events[len(events)-1].Sequence,
		SourceType: firstNonEmptyLogValue(map[bool]string{true: "kubernetes"}[sender.identity.ClusterID > 0], "file"), AssetType: sender.identity.AssetType, AssetID: sender.identity.AssetID,
		HostID: sender.identity.HostID, ClusterID: sender.identity.ClusterID,
		Environment: first.Environment, Service: first.Service, FilePath: first.FilePath,
		Stream: first.Stream, Namespace: first.Namespace, WorkloadKind: first.WorkloadKind,
		WorkloadName: first.WorkloadName, PodName: first.PodName, PodUID: first.PodUID,
		ContainerName: first.ContainerName, ContainerImage: first.ContainerImage,
		NodeName: firstNonEmptyLogValue(first.NodeName, sender.identity.NodeName), Records: make([]loggingest.LogRecord, 0, len(events)),
	}
	for _, event := range events {
		attributes := cloneStringMap(event.Attributes)
		attributes["log.source_id"] = event.SourceID
		batch.Records = append(batch.Records, loggingest.LogRecord{
			Sequence: event.Sequence, TimestampUnixNano: event.Timestamp.UnixNano(),
			ObservedTimestampUnixNano: event.ObservedAt.UnixNano(), Body: event.Body,
			SeverityText: event.Level, SeverityNumber: severityNumber(event.Level),
			TraceID: event.TraceID, SpanID: event.SpanID,
			RetentionDays: event.RetentionDays,
			Attributes:    attributes, ResourceAttributes: cloneStringMap(event.ResourceAttributes),
		})
	}
	return batch
}

func (sender *Sender) sendBatch(ctx context.Context, batch loggingest.LogBatch) error {
	raw, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, sender.gatewayURL+"/api/v1/logs/batches", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+sender.token)
	response, err := sender.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var ack loggingest.IngestAck
	if err := json.NewDecoder(response.Body).Decode(&ack); err != nil {
		return fmt.Errorf("解析 Gateway ACK 失败: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || ack.ErrorCode != "" {
		code := ack.ErrorCode
		if code == "" {
			code = fmt.Sprintf("HTTP_%d", response.StatusCode)
		}
		return &batchError{code: code, message: ack.ErrorMessage}
	}
	return nil
}

func groupEvents(events []Event) [][]Event {
	groups := make(map[string][]Event)
	order := make([]string, 0)
	for _, event := range events {
		key := fmt.Sprintf("%s\x00%d\x00%d\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s", event.SourceID, event.PolicyID, event.PolicyVersion, event.FilePath, event.Environment, event.Service, event.PodUID, event.ContainerName, event.Stream)
		if _, exists := groups[key]; !exists {
			order = append(order, key)
		}
		groups[key] = append(groups[key], event)
	}
	sort.SliceStable(order, func(left, right int) bool {
		return groups[order[left]][0].Sequence < groups[order[right]][0].Sequence
	})
	result := make([][]Event, 0, len(order))
	for _, key := range order {
		result = append(result, groups[key])
	}
	return result
}

func firstNonEmptyLogValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func deterministicBatchID(segment string, groupIndex int, sourceID string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s", segment, groupIndex, sourceID)))
	hash[6] = (hash[6] & 0x0f) | 0x40
	hash[8] = (hash[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		hash[0:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}

func severityNumber(level string) int32 {
	switch strings.ToUpper(level) {
	case "TRACE":
		return 1
	case "DEBUG":
		return 5
	case "INFO":
		return 9
	case "WARN", "WARNING":
		return 13
	case "ERROR":
		return 17
	case "FATAL":
		return 21
	default:
		return 0
	}
}

func cloneStringMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input)+1)
	for key, value := range input {
		result[key] = value
	}
	return result
}
