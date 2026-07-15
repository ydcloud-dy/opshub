package logagent

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type FileIdentity struct {
	Device uint64 `json:"device"`
	Inode  uint64 `json:"inode"`
}

type Checkpoint struct {
	SourceID    string       `json:"sourceId"`
	Path        string       `json:"path"`
	Identity    FileIdentity `json:"identity"`
	Fingerprint string       `json:"fingerprint"`
	Offset      int64        `json:"offset"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

type CheckpointStore struct {
	path        string
	mutex       sync.RWMutex
	checkpoints map[string]Checkpoint
}

func NewCheckpointStore(path string) (*CheckpointStore, error) {
	store := &CheckpointStore{path: path, checkpoints: make(map[string]Checkpoint)}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return store, nil
		}
		return nil, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &store.checkpoints); err != nil {
			return nil, fmt.Errorf("解析日志 checkpoint 失败: %w", err)
		}
	}
	return store, nil
}

func (store *CheckpointStore) Find(sourceID, path string, identity FileIdentity) (Checkpoint, bool) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	if checkpoint, exists := store.checkpoints[checkpointKey(sourceID, path)]; exists && checkpoint.Identity == identity {
		return checkpoint, true
	}
	for _, checkpoint := range store.checkpoints {
		if checkpoint.SourceID == sourceID && checkpoint.Identity == identity {
			return checkpoint, true
		}
	}
	return Checkpoint{}, false
}

func (store *CheckpointStore) Save(checkpoint Checkpoint) error {
	checkpoint.UpdatedAt = time.Now()
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.checkpoints[checkpointKey(checkpoint.SourceID, checkpoint.Path)] = checkpoint
	if err := os.MkdirAll(filepath.Dir(store.path), 0700); err != nil {
		return err
	}
	raw, err := json.Marshal(store.checkpoints)
	if err != nil {
		return err
	}
	temporary := store.path + ".tmp"
	if err := os.WriteFile(temporary, raw, 0600); err != nil {
		return err
	}
	return os.Rename(temporary, store.path)
}

func fileIdentity(info os.FileInfo) (FileIdentity, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return FileIdentity{}, fmt.Errorf("无法读取文件 inode")
	}
	return FileIdentity{Device: uint64(stat.Dev), Inode: uint64(stat.Ino)}, nil
}

func filePrefixFingerprint(file *os.File, size int64) (string, bool, error) {
	const fingerprintBytes = 64
	if size < fingerprintBytes {
		return "", false, nil
	}
	buffer := make([]byte, fingerprintBytes)
	if _, err := file.ReadAt(buffer, 0); err != nil {
		return "", false, err
	}
	hash := sha256.Sum256(buffer)
	return fmt.Sprintf("%x", hash[:]), true, nil
}

func checkpointKey(sourceID, path string) string {
	return sourceID + "\x00" + filepath.Clean(path)
}
