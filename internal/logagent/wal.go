package logagent

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var ErrWALFull = errors.New("日志 WAL 已达到容量上限")

type WAL struct {
	directory      string
	maxBytes       int64
	segmentRecords int
	metrics        *Metrics
	mutex          sync.Mutex
	activeFile     *os.File
	activePath     string
	activeRecords  int
	activeBytes    int64
	totalBytes     int64
	sequence       atomic.Uint64
}

func OpenWAL(directory string, maxBytes int64, segmentRecords int, metrics *Metrics) (*WAL, error) {
	if maxBytes <= 0 {
		maxBytes = defaultMaxWALBytes
	}
	if segmentRecords <= 0 || segmentRecords > 2000 {
		segmentRecords = defaultBatchSize
	}
	if metrics == nil {
		metrics = &Metrics{}
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	wal := &WAL{directory: directory, maxBytes: maxBytes, segmentRecords: segmentRecords, metrics: metrics}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".active") && !strings.HasSuffix(entry.Name(), ".ready")) {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		info, statErr := entry.Info()
		if statErr != nil {
			return nil, statErr
		}
		wal.totalBytes += info.Size()
		if strings.HasSuffix(entry.Name(), ".active") {
			readyPath := strings.TrimSuffix(path, ".active") + ".ready"
			if err := os.Rename(path, readyPath); err != nil {
				return nil, fmt.Errorf("恢复 WAL 活跃段失败: %w", err)
			}
		}
	}
	wal.metrics.walBytes.Store(wal.totalBytes)
	wal.sequence.Store(uint64(time.Now().UnixNano()))
	return wal, nil
}

func (wal *WAL) Append(event Event) error {
	wal.mutex.Lock()
	defer wal.mutex.Unlock()
	event.Sequence = wal.sequence.Add(1)
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	recordBytes := int64(len(raw) + 1)
	if wal.totalBytes+recordBytes > wal.maxBytes {
		return ErrWALFull
	}
	if wal.activeFile == nil {
		if err := wal.openActiveLocked(); err != nil {
			return err
		}
	}
	if _, err := wal.activeFile.Write(append(raw, '\n')); err != nil {
		return err
	}
	if err := wal.activeFile.Sync(); err != nil {
		return err
	}
	wal.activeRecords++
	wal.activeBytes += recordBytes
	wal.totalBytes += recordBytes
	wal.metrics.walBytes.Store(wal.totalBytes)
	if wal.activeRecords >= wal.segmentRecords {
		return wal.rotateLocked()
	}
	return nil
}

func (wal *WAL) Rotate() error {
	wal.mutex.Lock()
	defer wal.mutex.Unlock()
	return wal.rotateLocked()
}

func (wal *WAL) Close(rotate bool) error {
	wal.mutex.Lock()
	defer wal.mutex.Unlock()
	if wal.activeFile == nil {
		return nil
	}
	if rotate {
		return wal.rotateLocked()
	}
	err := wal.activeFile.Close()
	wal.activeFile = nil
	return err
}

func (wal *WAL) ReadySegments() ([]string, error) {
	entries, err := os.ReadDir(wal.directory)
	if err != nil {
		return nil, err
	}
	segments := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".ready") {
			segments = append(segments, filepath.Join(wal.directory, entry.Name()))
		}
	}
	sort.Strings(segments)
	return segments, nil
}

func (wal *WAL) ReadSegment(path string) ([]Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	events := make([]Event, 0, wal.segmentRecords)
	reader := bufio.NewReaderSize(file, 64*1024)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			var event Event
			if unmarshalErr := json.Unmarshal(line, &event); unmarshalErr != nil {
				return nil, fmt.Errorf("解析 WAL 段 %s 失败: %w", filepath.Base(path), unmarshalErr)
			}
			events = append(events, event)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return events, nil
}

func (wal *WAL) DeleteSegment(path string) error {
	info, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if info != nil {
		wal.mutex.Lock()
		wal.totalBytes -= info.Size()
		if wal.totalBytes < 0 {
			wal.totalBytes = 0
		}
		wal.metrics.walBytes.Store(wal.totalBytes)
		wal.mutex.Unlock()
	}
	return nil
}

func (wal *WAL) openActiveLocked() error {
	name := fmt.Sprintf("%020d.active", time.Now().UnixNano())
	path := filepath.Join(wal.directory, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	wal.activeFile = file
	wal.activePath = path
	wal.activeRecords = 0
	wal.activeBytes = 0
	return nil
}

func (wal *WAL) rotateLocked() error {
	if wal.activeFile == nil || wal.activeRecords == 0 {
		return nil
	}
	if err := wal.activeFile.Sync(); err != nil {
		return err
	}
	if err := wal.activeFile.Close(); err != nil {
		return err
	}
	readyPath := strings.TrimSuffix(wal.activePath, ".active") + ".ready"
	if err := os.Rename(wal.activePath, readyPath); err != nil {
		return err
	}
	wal.activeFile = nil
	wal.activePath = ""
	wal.activeRecords = 0
	wal.activeBytes = 0
	return nil
}
