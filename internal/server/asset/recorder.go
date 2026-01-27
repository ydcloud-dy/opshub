// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package asset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AsciinemaRecorder 实现终端录制功能，以asciinema格式保存
type AsciinemaRecorder struct {
	mu            sync.Mutex
	file          *os.File
	startTime     time.Time
	recordingPath string
	lastTime      float64
	cols          int
	rows          int
}

// AsciinemaHeader asciinema文件头部
type AsciinemaHeader struct {
	Version   int   `json:"version"`
	Width     int   `json:"width"`
	Height    int   `json:"height"`
	Timestamp int64 `json:"timestamp"`
}

// AsciinemaEvent asciinema事件
type AsciinemaEvent struct {
	Time float64
	Type string
	Data string
}

// NewAsciinemaRecorder 创建新的录制器
func NewAsciinemaRecorder(recordingDir string, cols, rows int) (*AsciinemaRecorder, error) {
	// 确保录制目录存在
	if err := os.MkdirAll(recordingDir, 0755); err != nil {
		return nil, fmt.Errorf("创建录制目录失败: %w", err)
	}

	// 生成录制文件名：时间戳.cast
	filename := time.Now().Format("20060102-150405") + ".cast"
	recordingPath := filepath.Join(recordingDir, filename)

	// 创建录制文件
	file, err := os.Create(recordingPath)
	if err != nil {
		return nil, fmt.Errorf("创建录制文件失败: %w", err)
	}

	recorder := &AsciinemaRecorder{
		file:          file,
		startTime:     time.Now(),
		recordingPath: recordingPath,
		lastTime:      0,
		cols:          cols,
		rows:          rows,
	}

	// 写入文件头部
	if err := recorder.writeHeader(); err != nil {
		file.Close()
		os.Remove(recordingPath)
		return nil, err
	}

	return recorder, nil
}

// writeHeader 写入asciinema文件头部
func (r *AsciinemaRecorder) writeHeader() error {
	header := AsciinemaHeader{
		Version:   2,
		Width:     r.cols,
		Height:    r.rows,
		Timestamp: r.startTime.Unix(),
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return fmt.Errorf("序列化头部失败: %w", err)
	}

	_, err = r.file.Write(append(headerBytes, '\n'))
	if err != nil {
		return fmt.Errorf("写入头部失败: %w", err)
	}

	return r.file.Sync()
}

// RecordOutput 记录终端输出
func (r *AsciinemaRecorder) RecordOutput(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.recordEvent("o", data)
}

// RecordInput 记录用户输入
func (r *AsciinemaRecorder) RecordInput(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.recordEvent("i", data)
}

// recordEvent 记录事件
func (r *AsciinemaRecorder) recordEvent(eventType string, data []byte) error {
	if r.file == nil {
		return fmt.Errorf("录制器已关闭")
	}

	// 计算相对时间（秒）
	currentTime := time.Since(r.startTime).Seconds()

	// 构建事件数组：[时间, 类型, 数据]
	event := []interface{}{currentTime, eventType, string(data)}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败: %w", err)
	}

	_, err = r.file.Write(append(eventBytes, '\n'))
	if err != nil {
		return fmt.Errorf("写入事件失败: %w", err)
	}

	r.lastTime = currentTime
	return nil
}

// Close 关闭录制器
func (r *AsciinemaRecorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file == nil {
		return nil
	}

	err := r.file.Sync()
	if err != nil {
		r.file.Close()
		return fmt.Errorf("同步文件失败: %w", err)
	}

	err = r.file.Close()
	r.file = nil
	return err
}

// GetRecordingPath 获取录制文件路径
func (r *AsciinemaRecorder) GetRecordingPath() string {
	return r.recordingPath
}

// GetDuration 获取录制时长（秒）
func (r *AsciinemaRecorder) GetDuration() int {
	return int(time.Since(r.startTime).Seconds())
}

// GetFileSize 获取文件大小（字节）
func (r *AsciinemaRecorder) GetFileSize() int64 {
	fileInfo, err := os.Stat(r.recordingPath)
	if err != nil {
		return 0
	}
	return fileInfo.Size()
}
