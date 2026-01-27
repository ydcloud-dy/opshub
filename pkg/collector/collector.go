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

package collector

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	sshclient "github.com/ydcloud-dy/opshub/pkg/ssh"
)

// SystemInfo 系统信息
type SystemInfo struct {
	OS       string      `json:"os"`       // 操作系统
	Kernel   string      `json:"kernel"`   // 内核版本
	Arch     string      `json:"arch"`     // 架构
	CPU      CPUInfo     `json:"cpu"`      // CPU信息
	Memory   MemoryInfo  `json:"memory"`   // 内存信息
	Disk     []DiskInfo  `json:"disk"`     // 磁盘信息
	Uptime   string      `json:"uptime"`   // 运行时间
	Hostname string      `json:"hostname"` // 主机名
}

// CPUInfo CPU信息
type CPUInfo struct {
	ModelName  string  `json:"modelName"`  // 型号
	Cores      int     `json:"cores"`      // 核心数
	Threads    int     `json:"threads"`    // 线程数
	Usage      float64 `json:"usage"`      // 使用率百分比
	MHz        float64 `json:"mHz"`        // 频率
	Cache      string  `json:"cache"`      // 缓存
	VendorID   string  `json:"vendorId"`   // 厂商
}

// ToJSON 转换为JSON
func (c CPUInfo) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MemoryInfo 内存信息
type MemoryInfo struct {
	Total     uint64  `json:"total"`     // 总内存(字节)
	Used      uint64  `json:"used"`      // 已用(字节)
	Free      uint64  `json:"free"`      // 空闲(字节)
	Available uint64  `json:"available"` // 可用(字节)
	Usage     float64 `json:"usage"`     // 使用率百分比
	SwapTotal uint64  `json:"swapTotal"` // 交换总空间
	SwapUsed  uint64  `json:"swapUsed"`  // 交换已用
}

// DiskInfo 磁盘信息
type DiskInfo struct {
	Device     string  `json:"device"`     // 设备
	MountPoint string  `json:"mountPoint"` // 挂载点
	Fstype     string  `json:"fstype"`     // 文件系统类型
	Total      uint64  `json:"total"`      // 总容量(字节)
	Used       uint64  `json:"used"`       // 已用(字节)
	Free       uint64  `json:"free"`       // 空闲(字节)
	Usage      float64 `json:"usage"`      // 使用率百分比
}

// Collector 采集器
type Collector struct {
	sshClient *sshclient.Client
	timeout   time.Duration
}

// NewCollector 创建采集器
func NewCollector(sshClient *sshclient.Client) *Collector {
	return &Collector{
		sshClient: sshClient,
		timeout:   30 * time.Second,
	}
}

// CollectAll 采集所有信息
func (c *Collector) CollectAll() (*SystemInfo, error) {
	info := &SystemInfo{}

	// 并发采集各个信息
	errChan := make(chan error, 5)
	var errCPU, errMem, errDisk, errSys, errUptime error

	go func() {
		info.CPU, errCPU = c.CollectCPU()
		errChan <- errCPU
	}()

	go func() {
		info.Memory, errMem = c.CollectMemory()
		errChan <- errMem
	}()

	go func() {
		info.Disk, errDisk = c.CollectDisk()
		errChan <- errDisk
	}()

	go func() {
		info.OS, info.Kernel, info.Arch, info.Hostname, errSys = c.CollectSystemInfo()
		errChan <- errSys
	}()

	go func() {
		info.Uptime, errUptime = c.CollectUptime()
		errChan <- errUptime
	}()

	// 等待所有采集完成
	for i := 0; i < 5; i++ {
		<-errChan
	}

	// 检查是否有严重错误
	if errCPU != nil && errMem != nil && errDisk != nil {
		return nil, fmt.Errorf("采集失败: CPU=%v, Memory=%v", errCPU, errMem)
	}

	return info, nil
}

// CollectSystemInfo 采集系统基本信息
func (c *Collector) CollectSystemInfo() (os, kernel, arch, hostname string, err error) {
	// 获取操作系统信息
	osOutput, err := c.sshClient.Execute(". /etc/os-release && echo $PRETTY_NAME")
	if err == nil && osOutput != "" {
		os = strings.TrimSpace(osOutput)
	} else {
		// 尝试其他方法
		osOutput, _ = c.sshClient.Execute("cat /etc/redhat-release 2>/dev/null || cat /etc/issue 2>/dev/null | head -1")
		os = strings.TrimSpace(osOutput)
	}

	// 获取内核版本
	kernelOutput, err := c.sshClient.Execute("uname -r")
	if err == nil {
		kernel = strings.TrimSpace(kernelOutput)
	}

	// 获取架构
	archOutput, err := c.sshClient.Execute("uname -m")
	if err == nil {
		arch = strings.TrimSpace(archOutput)
	}

	// 获取主机名
	hostnameOutput, err := c.sshClient.Execute("hostname")
	if err == nil {
		hostname = strings.TrimSpace(hostnameOutput)
	}

	return
}

// CollectCPU 采集CPU信息
func (c *Collector) CollectCPU() (CPUInfo, error) {
	info := CPUInfo{}

	// 获取CPU信息（兼容不同系统）
	cmd := "lscpu 2>/dev/null || cat /proc/cpuinfo"
	output, err := c.sshClient.Execute(cmd)
	if err != nil {
		return info, fmt.Errorf("获取CPU信息失败: %w", err)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "Model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.ModelName = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "CPU(s):") && !strings.Contains(line, "CPU MHz") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 && info.Threads == 0 {
				info.Threads, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
			}
		} else if strings.Contains(line, "Core(s) per socket:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cores, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				info.Cores = cores
			}
		} else if strings.Contains(line, "CPU MHz") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.MHz, _ = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			}
		} else if strings.Contains(line, "Vendor ID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.VendorID = strings.TrimSpace(parts[1])
			}
		}
	}

	// 计算CPU使用率
	usageOutput, err := c.sshClient.Execute("top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1")
	if err == nil {
		info.Usage, _ = strconv.ParseFloat(strings.TrimSpace(usageOutput), 64)
	}

	return info, nil
}

// CollectMemory 采集内存信息
func (c *Collector) CollectMemory() (MemoryInfo, error) {
	info := MemoryInfo{}

	output, err := c.sshClient.Execute("free -b")
	if err != nil {
		return info, fmt.Errorf("获取内存信息失败: %w", err)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "Mem:":
			if len(fields) >= 3 {
				info.Total, _ = strconv.ParseUint(fields[1], 10, 64)
				info.Used, _ = strconv.ParseUint(fields[2], 10, 64)
				info.Free, _ = strconv.ParseUint(fields[3], 10, 64)
			}
			if len(fields) >= 7 {
				info.Available, _ = strconv.ParseUint(fields[6], 10, 64)
			}
			if info.Total > 0 {
				info.Usage = float64(info.Used) / float64(info.Total) * 100
			}
		case "Swap:":
			if len(fields) >= 3 {
				info.SwapTotal, _ = strconv.ParseUint(fields[1], 10, 64)
				info.SwapUsed, _ = strconv.ParseUint(fields[2], 10, 64)
			}
		}
	}

	return info, nil
}

// CollectDisk 采集磁盘信息
func (c *Collector) CollectDisk() ([]DiskInfo, error) {
	// 只统计根目录 / 的磁盘信息
	output, err := c.sshClient.Execute("df -B1 /")
	if err != nil {
		return nil, fmt.Errorf("获取磁盘信息失败: %w", err)
	}

	var disks []DiskInfo
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		device := fields[0]
		mountPoint := fields[5]

		// 只处理根目录
		if mountPoint != "/" {
			continue
		}

		disk := DiskInfo{
			Device:     device,
			Total:      0,
			Used:       0,
			Free:       0,
			MountPoint: mountPoint,
		}

		disk.Total, _ = strconv.ParseUint(fields[1], 10, 64)
		disk.Used, _ = strconv.ParseUint(fields[2], 10, 64)
		disk.Free, _ = strconv.ParseUint(fields[3], 10, 64)

		if disk.Total > 0 {
			disk.Usage = float64(disk.Used) / float64(disk.Total) * 100
		}

		// 获取文件系统类型
		fstype := "unknown"
		if strings.Contains(disk.Device, "/dev/") || strings.Contains(disk.Device, "mapper") {
			fsOutput, _ := c.sshClient.Execute(fmt.Sprintf("blkid -o value -s TYPE %s 2>/dev/null", disk.Device))
			fstype = strings.TrimSpace(fsOutput)
			if fstype == "" {
				fstype = "ext4" // 默认
			}
		} else if strings.HasPrefix(disk.Device, "/") {
			fstype = "nfs"
		}
		disk.Fstype = fstype

		disks = append(disks, disk)
	}

	return disks, nil
}

// CollectProcessCount 采集进程数量
func (c *Collector) CollectProcessCount() (int, error) {
	output, err := c.sshClient.Execute("ps aux | wc -l")
	if err != nil {
		return 0, fmt.Errorf("获取进程数量失败: %w", err)
	}

	count, _ := strconv.Atoi(strings.TrimSpace(output))
	return count, nil
}

// CollectPortCount 采集端口监听数量
func (c *Collector) CollectPortCount() (int, error) {
	output, err := c.sshClient.Execute("ss -tlna 2>/dev/null | wc -l || netstat -tlna 2>/dev/null | wc -l")
	if err != nil {
		return 0, fmt.Errorf("获取端口数量失败: %w", err)
	}

	count, _ := strconv.Atoi(strings.TrimSpace(output))
	// 减去标题行
	if count > 0 {
		count--
	}
	return count, nil
}

// CollectUptime 采集运行时间
func (c *Collector) CollectUptime() (string, error) {
	output, err := c.sshClient.Execute("uptime -p 2>/dev/null || uptime")
	if err != nil {
		return "", fmt.Errorf("获取运行时间失败: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// ToJSON 转换为JSON
func (s *SystemInfo) ToJSON() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
