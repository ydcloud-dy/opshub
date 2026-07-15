package loggingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	logsvc "github.com/ydcloud-dy/opshub/plugins/logcenter/service"
)

type BatchSink interface {
	WriteBatch(ctx context.Context, batch LogBatch) error
}

type ClickHouseSink struct {
	cluster  logmodel.StorageCluster
	password string
	client   *logsvc.ClickHouseService
}

func NewClickHouseSink(cluster logmodel.StorageCluster, password string) *ClickHouseSink {
	return &ClickHouseSink{cluster: cluster, password: password, client: logsvc.NewClickHouseService()}
}

var storageIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func (s *ClickHouseSink) WriteBatch(ctx context.Context, batch LogBatch) error {
	database := strings.TrimSpace(s.cluster.DatabaseName)
	if database == "" {
		database = "opshub_logs"
	}
	if !storageIdentifierPattern.MatchString(database) {
		return fmt.Errorf("ClickHouse 数据库名称无效")
	}
	batchID := strings.TrimSpace(batch.BatchID)
	if !batchIDPattern.MatchString(batchID) {
		return fmt.Errorf("日志批次 ID 无效")
	}
	var body bytes.Buffer
	body.WriteString(fmt.Sprintf("INSERT INTO %s.opshub_logs SETTINGS insert_deduplication_token='%s', date_time_input_format='best_effort', input_format_defaults_for_omitted_fields=1 FORMAT JSONEachRow\n", database, batchID))
	encoder := json.NewEncoder(&body)
	for index, record := range batch.Records {
		sequence := record.Sequence
		if sequence == 0 {
			sequence = batch.SequenceStart + uint64(index)
		}
		observedAt := record.ObservedTimestampUnixNano
		if observedAt <= 0 {
			observedAt = record.TimestampUnixNano
		}
		row := map[string]interface{}{
			"tenant_id":           1,
			"timestamp":           formatClickHouseTime(record.TimestampUnixNano),
			"observed_at":         formatClickHouseTime(observedAt),
			"source_type":         batch.SourceType,
			"asset_type":          batch.AssetType,
			"asset_id":            batch.AssetID,
			"host_id":             batch.HostID,
			"cluster_id":          batch.ClusterID,
			"environment":         batch.Environment,
			"service":             batch.Service,
			"level":               normalizeSeverity(record.SeverityText),
			"retention_days":      normalizeRetentionDays(record.RetentionDays, s.cluster.DefaultRetentionDays),
			"namespace":           batch.Namespace,
			"workload_kind":       batch.WorkloadKind,
			"workload_name":       batch.WorkloadName,
			"pod_name":            batch.PodName,
			"pod_uid":             batch.PodUID,
			"container_name":      batch.ContainerName,
			"container_image":     batch.ContainerImage,
			"node_name":           batch.NodeName,
			"file_path":           batch.FilePath,
			"stream":              batch.Stream,
			"body":                record.Body,
			"attributes":          nonNilMap(record.Attributes),
			"resource_attributes": nonNilMap(record.ResourceAttributes),
			"trace_id":            record.TraceID,
			"span_id":             record.SpanID,
			"agent_id":            batch.AgentID,
			"policy_id":           batch.PolicyID,
			"policy_version":      batch.PolicyVersion,
			"batch_id":            batch.BatchID,
			"sequence":            sequence,
			"fingerprint":         fingerprint(batch, record),
		}
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}
	_, err := s.client.Execute(ctx, s.cluster, s.password, body.String(), nil)
	return err
}

func normalizeRetentionDays(value, fallback int) int {
	if value <= 0 {
		value = fallback
	}
	if value <= 0 {
		value = 30
	}
	if value > 3650 {
		value = 3650
	}
	return value
}

type WriterConfig struct {
	QueueCapacity  int
	Workers        int
	MaxRetries     int
	WriteTimeout   time.Duration
	DeadletterPath string
	DedupCapacity  int
	QueueMode      string
	QueueTopic     string
	ConsumerGroup  string
	BrokerCount    int
}

type writeJob struct {
	batch  LogBatch
	result chan IngestAck
	state  *batchState
}

type batchState struct {
	done chan struct{}
	ack  IngestAck
}

type Writer struct {
	config            WriterConfig
	sink              BatchSink
	queue             chan writeJob
	startedAt         time.Time
	stop              chan struct{}
	stopOnce          sync.Once
	waitGroup         sync.WaitGroup
	dedupMu           sync.Mutex
	dedup             map[string]time.Time
	dedupOrder        []string
	inflight          map[string]*batchState
	acceptedBatches   atomic.Uint64
	acceptedRecords   atomic.Uint64
	rejectedBatches   atomic.Uint64
	duplicateBatches  atomic.Uint64
	failedBatches     atomic.Uint64
	deadletterBatches atomic.Uint64
	writeNanos        atomic.Uint64
	writeCount        atomic.Uint64
	queueLag          atomic.Int64
	queueHealthy      atomic.Bool
	queueErrorMu      sync.RWMutex
	queueLastError    string
	lastMu            sync.RWMutex
	lastSuccessAt     *time.Time
	lastErrorAt       *time.Time
	lastError         string
}

func NewWriter(config WriterConfig, sink BatchSink) *Writer {
	if config.QueueCapacity <= 0 {
		config.QueueCapacity = 1024
	}
	if config.Workers <= 0 {
		config.Workers = 4
	}
	if config.MaxRetries < 0 {
		config.MaxRetries = 0
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = 15 * time.Second
	}
	if config.DedupCapacity <= 0 {
		config.DedupCapacity = 100000
	}
	config.QueueMode = strings.ToLower(strings.TrimSpace(config.QueueMode))
	if config.QueueMode == "" {
		config.QueueMode = "direct"
	}
	writer := &Writer{
		config:    config,
		sink:      sink,
		queue:     make(chan writeJob, config.QueueCapacity),
		startedAt: time.Now(),
		stop:      make(chan struct{}),
		dedup:     make(map[string]time.Time),
		inflight:  make(map[string]*batchState),
	}
	writer.queueHealthy.Store(config.QueueMode == "direct")
	for index := 0; index < config.Workers; index++ {
		writer.waitGroup.Add(1)
		go writer.runWorker()
	}
	return writer
}

func (w *Writer) Submit(ctx context.Context, batch LogBatch) IngestAck {
	state, completed, owner := w.beginBatch(batch.BatchID)
	if completed {
		w.duplicateBatches.Add(1)
		return successAck(batch, true)
	}
	if !owner {
		select {
		case <-state.done:
			ack := state.ack
			if ack.ErrorCode == "" {
				ack.Duplicate = true
				w.duplicateBatches.Add(1)
			}
			return ack
		case <-ctx.Done():
			return errorAck(batch.BatchID, "WRITE_TIMEOUT", ctx.Err().Error(), 1000)
		}
	}
	job := writeJob{batch: batch, result: make(chan IngestAck, 1), state: state}
	select {
	case w.queue <- job:
	case <-ctx.Done():
		w.rejectedBatches.Add(1)
		ack := errorAck(batch.BatchID, "QUEUE_TIMEOUT", ctx.Err().Error(), 1000)
		w.finishBatch(batch.BatchID, state, ack)
		return ack
	default:
		w.rejectedBatches.Add(1)
		ack := errorAck(batch.BatchID, "QUEUE_FULL", "Writer 队列已满", 1000)
		w.finishBatch(batch.BatchID, state, ack)
		return ack
	}
	select {
	case result := <-job.result:
		return result
	case <-ctx.Done():
		return errorAck(batch.BatchID, "WRITE_TIMEOUT", ctx.Err().Error(), 1000)
	}
}

func (w *Writer) Close() {
	w.stopOnce.Do(func() { close(w.stop) })
	w.waitGroup.Wait()
}

func (w *Writer) Status() ComponentStatus {
	w.lastMu.RLock()
	defer w.lastMu.RUnlock()
	w.queueErrorMu.RLock()
	queueLastError := w.queueLastError
	w.queueErrorMu.RUnlock()
	status := "healthy"
	if !w.queueHealthy.Load() {
		status = "degraded"
	}
	return ComponentStatus{
		Name: "log-writer", InstanceID: componentInstanceID(), Status: status,
		QueueMode: w.config.QueueMode, QueueTopic: w.config.QueueTopic, ConsumerGroup: w.config.ConsumerGroup,
		BrokerCount: w.config.BrokerCount, QueueHealthy: w.queueHealthy.Load(), QueueLag: w.queueLag.Load(),
		StartedAt:       w.startedAt,
		UptimeSeconds:   int64(time.Since(w.startedAt).Seconds()),
		AcceptedBatches: w.acceptedBatches.Load(), AcceptedRecords: w.acceptedRecords.Load(),
		RejectedBatches: w.rejectedBatches.Load(), DuplicateBatches: w.duplicateBatches.Load(),
		FailedBatches: w.failedBatches.Load(), QueueDepth: len(w.queue), QueueCapacity: cap(w.queue),
		WriteLatencyMS:    averageMilliseconds(w.writeNanos.Load(), w.writeCount.Load()),
		DeadletterBatches: w.deadletterBatches.Load(), QueueLastError: queueLastError,
		LastSuccessAt: w.lastSuccessAt, LastErrorAt: w.lastErrorAt, LastError: w.lastError,
	}
}

func (w *Writer) SetQueueState(healthy bool, lag int64, err error) {
	w.queueHealthy.Store(healthy)
	if lag >= 0 {
		w.queueLag.Store(lag)
	}
	w.queueErrorMu.Lock()
	if err == nil {
		w.queueLastError = ""
	} else {
		w.queueLastError = err.Error()
	}
	w.queueErrorMu.Unlock()
}

func (w *Writer) MarkDeadletter() {
	w.deadletterBatches.Add(1)
}

func (w *Writer) runWorker() {
	defer w.waitGroup.Done()
	for {
		select {
		case <-w.stop:
			return
		case job := <-w.queue:
			ack := w.writeWithRetry(job.batch)
			w.finishBatch(job.batch.BatchID, job.state, ack)
			job.result <- ack
		}
	}
}

func (w *Writer) writeWithRetry(batch LogBatch) IngestAck {
	started := time.Now()
	defer func() {
		w.writeNanos.Add(uint64(time.Since(started)))
		w.writeCount.Add(1)
	}()
	var lastErr error
	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), w.config.WriteTimeout)
		lastErr = w.sink.WriteBatch(ctx, batch)
		cancel()
		if lastErr == nil {
			w.acceptedBatches.Add(1)
			w.acceptedRecords.Add(uint64(len(batch.Records)))
			now := time.Now()
			w.lastMu.Lock()
			w.lastSuccessAt = &now
			w.lastError = ""
			w.lastMu.Unlock()
			return successAck(batch, false)
		}
		if attempt < w.config.MaxRetries {
			time.Sleep(time.Duration(1<<attempt) * 100 * time.Millisecond)
		}
	}
	w.failedBatches.Add(1)
	now := time.Now()
	w.lastMu.Lock()
	w.lastErrorAt = &now
	w.lastError = lastErr.Error()
	w.lastMu.Unlock()
	if w.config.QueueMode == "direct" {
		w.writeDeadletter(batch, lastErr)
	}
	return errorAck(batch.BatchID, "CLICKHOUSE_WRITE_FAILED", lastErr.Error(), 2000)
}

func (w *Writer) beginBatch(batchID string) (*batchState, bool, bool) {
	w.dedupMu.Lock()
	defer w.dedupMu.Unlock()
	if _, exists := w.dedup[batchID]; exists {
		return nil, true, false
	}
	if state, exists := w.inflight[batchID]; exists {
		return state, false, false
	}
	state := &batchState{done: make(chan struct{})}
	w.inflight[batchID] = state
	return state, false, true
}

func (w *Writer) finishBatch(batchID string, state *batchState, ack IngestAck) {
	w.dedupMu.Lock()
	defer w.dedupMu.Unlock()
	current, exists := w.inflight[batchID]
	if !exists || current != state {
		return
	}
	state.ack = ack
	delete(w.inflight, batchID)
	if ack.ErrorCode == "" {
		w.dedup[batchID] = time.Now()
		w.dedupOrder = append(w.dedupOrder, batchID)
		if len(w.dedupOrder) > w.config.DedupCapacity {
			oldest := w.dedupOrder[0]
			w.dedupOrder = w.dedupOrder[1:]
			delete(w.dedup, oldest)
		}
	}
	close(state.done)
}

func (w *Writer) writeDeadletter(batch LogBatch, writeErr error) {
	if strings.TrimSpace(w.config.DeadletterPath) == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(w.config.DeadletterPath), 0o755)
	file, err := os.OpenFile(w.config.DeadletterPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	if json.NewEncoder(file).Encode(map[string]interface{}{"failedAt": time.Now().UTC(), "error": writeErr.Error(), "batch": batch}) == nil {
		w.MarkDeadletter()
	}
}

func averageMilliseconds(totalNanos, count uint64) float64 {
	if count == 0 {
		return 0
	}
	return float64(totalNanos) / float64(count) / float64(time.Millisecond)
}

func boolMetric(value bool) int {
	if value {
		return 1
	}
	return 0
}

func componentInstanceID() string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return "unknown"
	}
	return hostname
}

func successAck(batch LogBatch, duplicate bool) IngestAck {
	sequence := batch.SequenceEnd
	if sequence == 0 && len(batch.Records) > 0 {
		sequence = batch.SequenceStart + uint64(len(batch.Records)-1)
	}
	return IngestAck{BatchID: batch.BatchID, AcceptedRecords: len(batch.Records), AcceptedSequence: sequence, Duplicate: duplicate}
}

func errorAck(batchID, code, message string, retryAfter int) IngestAck {
	return IngestAck{BatchID: batchID, ErrorCode: code, ErrorMessage: message, RetryAfterMS: retryAfter}
}

func formatClickHouseTime(nanos int64) string {
	return time.Unix(0, nanos).UTC().Format("2006-01-02 15:04:05.000000000")
}

func normalizeSeverity(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "INFO"
	}
	return value
}

func nonNilMap(value map[string]string) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	return value
}

func fingerprint(batch LogBatch, record LogRecord) uint64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(batch.AssetType))
	_, _ = hash.Write([]byte(fmt.Sprintf("%d", batch.AssetID)))
	_, _ = hash.Write([]byte(batch.FilePath))
	_, _ = hash.Write([]byte(batch.ContainerName))
	_, _ = hash.Write([]byte(record.Body))
	return hash.Sum64()
}
