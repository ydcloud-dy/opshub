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

package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AsciinemaRecorder 终端会话录制器
type AsciinemaRecorder struct {
	mu             sync.Mutex
	file           *os.File
	startTime      time.Time
	recordingPath  string
	lastTime       float64
	cols           int
	rows           int
}

// AsciinemaHeader asciinema 格式头部
type AsciinemaHeader struct {
	Version   int     `json:"version"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Timestamp float64 `json:"timestamp"`
}

// AsciinemaEvent asciinema 事件
// 格式: [时间, 类型, 数据]
// 类型: "o" = 输出, "i" = 输入
type AsciinemaEvent []interface{}

// NewAsciinemaRecorder 创建录制器
func NewAsciinemaRecorder(recordingDir string, cols, rows int) (*AsciinemaRecorder, error) {
	// 确保录制目录存在
	if err := os.MkdirAll(recordingDir, 0755); err != nil {
		return nil, fmt.Errorf("创建录制目录失败: %w", err)
	}

	// 生成文件名
	filename := filepath.Join(recordingDir,
		fmt.Sprintf("%s.cast", time.Now().Format("20060102-150405")))

	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("创建录制文件失败: %w", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	recorder := &AsciinemaRecorder{
		file:          file,
		startTime:     time.Now(),
		recordingPath: filename,
		lastTime:      0,
		cols:          cols,
		rows:          rows,
	}

	// 写入头部
	header := AsciinemaHeader{
		Version:   2,
		Width:     cols,
		Height:    rows,
		Timestamp: now,
	}

	headerData, err := json.Marshal(header)
	if err != nil {
		file.Close()
		os.Remove(filename)
		return nil, fmt.Errorf("序列化头部失败: %w", err)
	}

	if _, err := file.Write(append(headerData, '\n')); err != nil {
		file.Close()
		os.Remove(filename)
		return nil, fmt.Errorf("写入头部失败: %w", err)
	}

	return recorder, nil
}

// RecordOutput 记录输出事件
func (r *AsciinemaRecorder) RecordOutput(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	elapsed := float64(time.Since(r.startTime).Microseconds()) / 1e6
	if elapsed <= r.lastTime {
		elapsed = r.lastTime + 0.0001
	}
	r.lastTime = elapsed

	// 将数据转换为可打印的字符串，替换不可打印字符
	dataStr := string(data)
	dataStr = sanitizeString(dataStr)

	event := AsciinemaEvent{elapsed, "o", dataStr}
	return r.writeEvent(event)
}

// RecordInput 记录输入事件
func (r *AsciinemaRecorder) RecordInput(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	elapsed := float64(time.Since(r.startTime).Microseconds()) / 1e6
	if elapsed <= r.lastTime {
		elapsed = r.lastTime + 0.0001
	}
	r.lastTime = elapsed

	dataStr := string(data)
	dataStr = sanitizeString(dataStr)

	event := AsciinemaEvent{elapsed, "i", dataStr}
	return r.writeEvent(event)
}

// writeEvent 写入事件到文件
func (r *AsciinemaRecorder) writeEvent(event AsciinemaEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = r.file.Write(data)
	return err
}

// Close 关闭录制器并保存
func (r *AsciinemaRecorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// GetRecordingPath 获取录制文件路径
func (r *AsciinemaRecorder) GetRecordingPath() string {
	return r.recordingPath
}

// GetDuration 获取录制时长（秒）
func (r *AsciinemaRecorder) GetDuration() int {
	return int(time.Since(r.startTime).Seconds())
}

// GetFileSize 获取文件大小
func (r *AsciinemaRecorder) GetFileSize() int64 {
	info, err := os.Stat(r.recordingPath)
	if err != nil {
		return 0
	}
	return info.Size()
}

// sanitizeString 清理字符串，确保可以正确序列化为JSON
func sanitizeString(s string) string {
	// 直接返回原始字符串
	// Go 的 json.Marshal 会自动处理所有必要的转义
	// 包括 ANSI 转义序列（ESC 字符）
	return s
}
