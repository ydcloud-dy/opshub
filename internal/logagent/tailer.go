package logagent

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type sourceRuntime struct {
	config   SourceConfig
	parser   LineParser
	redactor *Redactor
}

type trackedFile struct {
	source          *sourceRuntime
	path            string
	identity        FileIdentity
	file            *os.File
	reader          *bufio.Reader
	assembler       *MultilineAssembler
	cri             *CRIAssembler
	pendingMeta     CRIRecord
	hasPendingMeta  bool
	offset          int64
	committedOffset int64
	pendingOffset   int64
	fingerprint     string
	missingScans    int
	readAny         bool
}

type Tailer struct {
	config      Config
	checkpoints *CheckpointStore
	sink        EventSink
	metrics     *Metrics
	sources     []*sourceRuntime
	files       map[string]*trackedFile
	metadata    KubernetesMetadataResolver
}

type TailerOptions struct {
	KubernetesMetadataResolver KubernetesMetadataResolver
}

func NewTailer(config Config, checkpoints *CheckpointStore, sink EventSink, metrics *Metrics) (*Tailer, error) {
	return NewTailerWithOptions(config, checkpoints, sink, metrics, TailerOptions{})
}

func NewTailerWithOptions(config Config, checkpoints *CheckpointStore, sink EventSink, metrics *Metrics, options TailerOptions) (*Tailer, error) {
	config.Normalize()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if metrics == nil {
		metrics = &Metrics{}
	}
	tailer := &Tailer{
		config: config, checkpoints: checkpoints, sink: sink, metrics: metrics,
		files: make(map[string]*trackedFile), metadata: options.KubernetesMetadataResolver,
	}
	for index := range config.Sources {
		parser, err := NewLineParser(config.Sources[index].Parser)
		if err != nil {
			return nil, fmt.Errorf("初始化日志源 %s 解析器失败: %w", config.Sources[index].ID, err)
		}
		redactor, err := NewRedactor(config.Sources[index].Redaction)
		if err != nil {
			return nil, fmt.Errorf("初始化日志源 %s 脱敏器失败: %w", config.Sources[index].ID, err)
		}
		tailer.sources = append(tailer.sources, &sourceRuntime{config: config.Sources[index], parser: parser, redactor: redactor})
	}
	return tailer, nil
}

func (tailer *Tailer) Run(ctx context.Context) error {
	ticker := time.NewTicker(tailer.config.ScanInterval())
	defer ticker.Stop()
	defer tailer.closeAll()
	for {
		if err := tailer.ScanOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			tailer.metrics.recordError(err)
		}
		select {
		case <-ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			tailer.flushAll(flushCtx)
			cancel()
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (tailer *Tailer) ScanOnce(ctx context.Context) error {
	discovered := make(map[string]struct{})
	for _, source := range tailer.sources {
		paths, err := discoverPaths(source.config)
		if err != nil {
			return err
		}
		for _, path := range paths {
			info, err := os.Stat(path)
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			identity, err := fileIdentity(info)
			if err != nil {
				continue
			}
			key := trackedFileKey(source.config.ID, identity)
			discovered[key] = struct{}{}
			if existing := tailer.files[key]; existing != nil {
				existing.path = path
				existing.missingScans = 0
				continue
			}
			tracked, err := tailer.openTrackedFile(source, path, info, identity)
			if err != nil {
				return err
			}
			tailer.files[key] = tracked
		}
	}

	keys := make([]string, 0, len(tailer.files))
	for key := range tailer.files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		tracked := tailer.files[key]
		if _, exists := discovered[key]; !exists {
			tracked.missingScans++
		}
		if err := tailer.readAvailable(ctx, tracked); err != nil {
			return err
		}
		if tracked.missingScans >= 2 && !tracked.readAny {
			tailer.flushTracked(ctx, tracked)
			_ = tracked.file.Close()
			delete(tailer.files, key)
		}
		tracked.readAny = false
	}
	return nil
}

func (tailer *Tailer) openTrackedFile(source *sourceRuntime, path string, info os.FileInfo, identity FileIdentity) (*trackedFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	offset := int64(0)
	fingerprint, hasFingerprint, err := filePrefixFingerprint(file, info.Size())
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if checkpoint, exists := tailer.checkpoints.Find(source.config.ID, path, identity); exists {
		offset = checkpoint.Offset
		if offset > info.Size() || (hasFingerprint && checkpoint.Fingerprint != "" && checkpoint.Fingerprint != fingerprint) {
			offset = 0
		}
	} else if source.config.ReadFrom == "latest" {
		offset = info.Size()
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, err
	}
	assembler, err := NewMultilineAssembler(source.config.Multiline)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	tracked := &trackedFile{
		source: source, path: path, identity: identity, file: file,
		reader: bufio.NewReaderSize(file, 64*1024), assembler: assembler,
		offset: offset, committedOffset: offset, pendingOffset: offset, fingerprint: fingerprint,
	}
	if source.config.Format == "cri" {
		tracked.cri = NewCRIAssembler(source.config.MaxLineBytes)
	}
	return tracked, nil
}

func (tailer *Tailer) readAvailable(ctx context.Context, tracked *trackedFile) error {
	info, err := tracked.file.Stat()
	if err != nil {
		return err
	}
	if info.Size() < tracked.offset {
		if err := tracked.reset(0); err != nil {
			return err
		}
		tracked.committedOffset = 0
		tracked.pendingOffset = 0
	}
	currentFingerprint, hasFingerprint, err := filePrefixFingerprint(tracked.file, info.Size())
	if err != nil {
		return err
	}
	if hasFingerprint && tracked.fingerprint != "" && currentFingerprint != tracked.fingerprint && tracked.committedOffset > 0 {
		if err := tracked.reset(0); err != nil {
			return err
		}
		tracked.committedOffset = 0
		tracked.pendingOffset = 0
	}
	if hasFingerprint {
		tracked.fingerprint = currentFingerprint
	}
	for records := 0; records < 5000; records++ {
		line, consumed, truncated, err := readBoundedLine(tracked.reader, tracked.source.config.MaxLineBytes)
		if errors.Is(err, io.EOF) && consumed == 0 {
			if err := tailer.flushExpired(ctx, tracked); err != nil {
				return tailer.rewind(tracked, err)
			}
			return nil
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		tracked.readAny = true
		tracked.offset += consumed
		logicalRecords := []CRIRecord{{Body: line, Stream: tracked.source.config.Stream}}
		if tracked.cri != nil {
			logicalRecords = tracked.cri.Add(line, time.Now())
			if truncated && len(logicalRecords) == 0 {
				tracked.cri.MarkTruncated()
			}
		}
		for _, logicalRecord := range logicalRecords {
			if err := tailer.processRecord(ctx, tracked, logicalRecord, tracked.offset, truncated); err != nil {
				return tailer.rewind(tracked, err)
			}
		}
		if len(logicalRecords) == 0 {
			tracked.pendingOffset = tracked.offset
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
	}
	return nil
}

func (tailer *Tailer) processRecord(ctx context.Context, tracked *trackedFile, record CRIRecord, offset int64, truncated bool) error {
	if !tracked.source.config.Multiline.Enabled {
		return tailer.persist(ctx, tracked, record, record.Body, offset, truncated)
	}
	previousOffset := tracked.pendingOffset
	if !tracked.hasPendingMeta {
		tracked.pendingMeta = record
		tracked.hasPendingMeta = true
	}
	values := tracked.assembler.Add(record.Body, time.Now())
	for _, value := range values {
		meta := tracked.pendingMeta
		if err := tailer.persist(ctx, tracked, meta, value, previousOffset, truncated); err != nil {
			return err
		}
		tracked.pendingMeta = record
		tracked.hasPendingMeta = true
	}
	tracked.pendingOffset = offset
	return nil
}

func (tailer *Tailer) persist(ctx context.Context, tracked *trackedFile, runtimeRecord CRIRecord, line string, offset int64, truncated bool) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	observedAt := time.Now()
	parseTime := runtimeRecord.Timestamp
	if parseTime.IsZero() {
		parseTime = observedAt
	}
	parsed, err := tracked.source.parser.Parse(line, parseTime)
	if err != nil {
		tailer.metrics.parseErrors.Add(1)
		traceID, spanID := extractTraceContext(line)
		parsed = ParsedRecord{
			Timestamp: parseTime, Body: line, Level: detectLevel(line), Attributes: map[string]string{"log.parser_error": err.Error()},
			TraceID: traceID, SpanID: spanID,
		}
	}
	if parsed.Attributes == nil {
		parsed.Attributes = make(map[string]string)
	}
	for key, value := range runtimeRecord.Attributes {
		parsed.Attributes[key] = value
	}
	if truncated || parsed.Attributes["log.truncated"] == "true" {
		parsed.Attributes["log.truncated"] = "true"
		tailer.metrics.truncated.Add(1)
	}
	event := Event{
		SourceID: tracked.source.config.ID, PolicyID: tracked.source.config.PolicyID,
		PolicyVersion: tracked.source.config.PolicyVersion, FilePath: tracked.path,
		Environment: tracked.source.config.Environment, Service: tracked.source.config.Service,
		Stream: firstNonEmptyLogValue(runtimeRecord.Stream, tracked.source.config.Stream), Timestamp: parsed.Timestamp, ObservedAt: observedAt,
		Body: parsed.Body, Level: parsed.Level, TraceID: parsed.TraceID, SpanID: parsed.SpanID, Attributes: parsed.Attributes,
		RetentionDays: tracked.source.config.Retention.DaysForLevel(parsed.Level),
	}
	if tracked.source.config.Kubernetes != nil {
		metadata, selected, err := tailer.metadata.Resolve(tracked.path, *tracked.source.config.Kubernetes)
		if err != nil {
			return err
		}
		if !selected {
			return tailer.commitCheckpoint(tracked, offset)
		}
		applyKubernetesMetadata(&event, metadata)
	}
	tracked.source.redactor.Apply(&event)
	if err := tailer.sink.Append(event); err != nil {
		return err
	}
	if err := tailer.commitCheckpoint(tracked, offset); err != nil {
		return err
	}
	tailer.metrics.inputRecords.Add(1)
	return nil
}

func (tailer *Tailer) commitCheckpoint(tracked *trackedFile, offset int64) error {
	checkpoint := Checkpoint{
		SourceID: tracked.source.config.ID, Path: tracked.path, Identity: tracked.identity,
		Fingerprint: tracked.fingerprint, Offset: offset,
	}
	if err := tailer.checkpoints.Save(checkpoint); err != nil {
		return err
	}
	tracked.committedOffset = offset
	return nil
}

func applyKubernetesMetadata(event *Event, metadata KubernetesMetadata) {
	event.Namespace = metadata.Namespace
	event.WorkloadKind = metadata.WorkloadKind
	event.WorkloadName = metadata.WorkloadName
	event.PodName = metadata.PodName
	event.PodUID = metadata.PodUID
	event.ContainerName = metadata.ContainerName
	event.ContainerImage = metadata.ContainerImage
	event.NodeName = metadata.NodeName
	event.Environment = firstNonEmptyLogValue(metadata.Environment, event.Environment)
	event.Service = firstNonEmptyLogValue(metadata.Service, event.Service)
	event.ResourceAttributes = cloneStringMap(metadata.ResourceAttributes)
	if event.ResourceAttributes == nil {
		event.ResourceAttributes = make(map[string]string)
	}
	for key, value := range metadata.Labels {
		event.ResourceAttributes["k8s.label."+key] = value
	}
	for key, value := range metadata.Annotations {
		event.ResourceAttributes["k8s.annotation."+key] = value
	}
}

func (tailer *Tailer) rewind(tracked *trackedFile, cause error) error {
	if err := tracked.reset(tracked.committedOffset); err != nil {
		return fmt.Errorf("%v；回退日志读取位置失败: %w", cause, err)
	}
	tracked.pendingOffset = tracked.committedOffset
	return cause
}

func (tracked *trackedFile) reset(offset int64) error {
	if _, err := tracked.file.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	tracked.reader.Reset(tracked.file)
	tracked.assembler.Reset()
	if tracked.cri != nil {
		tracked.cri.Reset()
	}
	tracked.pendingMeta = CRIRecord{}
	tracked.hasPendingMeta = false
	tracked.offset = offset
	return nil
}

func (tailer *Tailer) flushTracked(ctx context.Context, tracked *trackedFile) {
	if tracked.cri != nil {
		for _, record := range tracked.cri.Flush() {
			if err := tailer.processRecord(ctx, tracked, record, tracked.pendingOffset, false); err != nil {
				tailer.metrics.recordError(err)
			}
		}
	}
	for _, value := range tracked.assembler.Flush() {
		meta := tracked.pendingMeta
		if err := tailer.persist(ctx, tracked, meta, value, tracked.pendingOffset, false); err != nil {
			tailer.metrics.recordError(err)
		}
		tracked.pendingMeta = CRIRecord{}
		tracked.hasPendingMeta = false
	}
}

func (tailer *Tailer) flushExpired(ctx context.Context, tracked *trackedFile) error {
	for _, value := range tracked.assembler.FlushExpired(time.Now()) {
		if err := tailer.persist(ctx, tracked, tracked.pendingMeta, value, tracked.pendingOffset, false); err != nil {
			return err
		}
		tracked.pendingMeta = CRIRecord{}
		tracked.hasPendingMeta = false
	}
	return nil
}

func (tailer *Tailer) flushAll(ctx context.Context) {
	for _, tracked := range tailer.files {
		tailer.flushTracked(ctx, tracked)
	}
}

func (tailer *Tailer) closeAll() {
	for key, tracked := range tailer.files {
		_ = tracked.file.Close()
		delete(tailer.files, key)
	}
}

func discoverPaths(source SourceConfig) ([]string, error) {
	unique := make(map[string]struct{})
	for _, pattern := range source.Paths {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("展开日志路径 %s 失败: %w", pattern, err)
		}
		for _, path := range matches {
			if !excludedPath(path, source.ExcludePaths) {
				unique[filepath.Clean(path)] = struct{}{}
			}
		}
	}
	paths := make([]string, 0, len(unique))
	for path := range unique {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func excludedPath(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}
	return false
}

func trackedFileKey(sourceID string, identity FileIdentity) string {
	return fmt.Sprintf("%s:%d:%d", sourceID, identity.Device, identity.Inode)
}

func readBoundedLine(reader *bufio.Reader, limit int) (string, int64, bool, error) {
	if limit <= 0 {
		limit = defaultMaxLineBytes
	}
	var output bytes.Buffer
	consumed := int64(0)
	truncated := false
	for {
		fragment, err := reader.ReadSlice('\n')
		consumed += int64(len(fragment))
		remaining := limit - output.Len()
		if remaining > 0 {
			if len(fragment) > remaining {
				output.Write(fragment[:remaining])
				truncated = true
			} else {
				output.Write(fragment)
			}
		} else if len(fragment) > 0 {
			truncated = true
		}
		if err == nil {
			break
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		value := strings.TrimSuffix(strings.TrimSuffix(output.String(), "\n"), "\r")
		return value, consumed, truncated, err
	}
	value := strings.TrimSuffix(strings.TrimSuffix(output.String(), "\n"), "\r")
	return value, consumed, truncated, nil
}
