//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const defaultVersion = "0.1.0"

var (
	version = defaultVersion
)

type config struct {
	ServerURL       string `json:"serverUrl"`
	EnrollmentToken string `json:"enrollmentToken"`
	AgentID         string `json:"agentId"`
	AgentToken      string `json:"agentToken"`
	Interval        int    `json:"interval"`
}

type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type registerResponse struct {
	AgentID    string `json:"agentId"`
	AgentToken string `json:"agentToken"`
	HostID     uint   `json:"hostId"`
	Interval   int    `json:"interval"`
}

type metricsPayload struct {
	AgentID     string  `json:"agentId"`
	AgentToken  string  `json:"agentToken"`
	Hostname    string  `json:"hostname"`
	IP          string  `json:"ip"`
	Version     string  `json:"version"`
	OS          string  `json:"os"`
	Kernel      string  `json:"kernel"`
	Arch        string  `json:"arch"`
	CPUCores    int     `json:"cpuCores"`
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryTotal uint64  `json:"memoryTotal"`
	MemoryUsed  uint64  `json:"memoryUsed"`
	MemoryUsage float64 `json:"memoryUsage"`
	DiskTotal   uint64  `json:"diskTotal"`
	DiskUsed    uint64  `json:"diskUsed"`
	DiskUsage   float64 `json:"diskUsage"`
	Uptime      string  `json:"uptime"`
}

func main() {
	var (
		configPath      string
		serverURL       string
		enrollmentToken string
		interval        int
		once            bool
		printVersion    bool
	)

	flag.StringVar(&configPath, "config", "/etc/opshub-agent/agent.json", "Agent配置文件")
	flag.StringVar(&serverURL, "server", "", "OpsHub服务地址")
	flag.StringVar(&enrollmentToken, "enrollment-token", "", "Agent安装注册令牌")
	flag.IntVar(&interval, "interval", 30, "采集间隔秒数")
	flag.BoolVar(&once, "once", false, "仅采集上报一次")
	flag.BoolVar(&printVersion, "version", false, "显示版本")
	flag.Parse()

	if printVersion {
		fmt.Printf("opshub-agent %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}

	cfg, err := loadOrInitConfig(configPath, serverURL, enrollmentToken, interval)
	if err != nil {
		fatal(err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	if err := ensureRegistered(client, configPath, cfg); err != nil {
		fatal(err)
	}

	if once {
		if err := collectAndReport(client, cfg); err != nil {
			fatal(err)
		}
		return
	}

	for {
		if err := collectAndReport(client, cfg); err != nil {
			logf("采集或上报失败: %v", err)
		}
		time.Sleep(time.Duration(cfg.Interval) * time.Second)
	}
}

func loadOrInitConfig(path, serverURL, enrollmentToken string, interval int) (*config, error) {
	cfg, err := loadConfig(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if cfg == nil {
		cfg = &config{}
	}
	if serverURL != "" {
		cfg.ServerURL = strings.TrimRight(serverURL, "/")
	}
	if enrollmentToken != "" {
		cfg.EnrollmentToken = enrollmentToken
	}
	if interval > 0 {
		cfg.Interval = interval
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 30
	}
	if cfg.ServerURL == "" {
		return nil, fmt.Errorf("缺少OpsHub服务地址，请指定 --server")
	}
	if cfg.AgentID == "" && cfg.EnrollmentToken == "" {
		return nil, fmt.Errorf("缺少Agent注册令牌，请指定 --enrollment-token")
	}
	if err := saveConfig(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadConfig(path string) (*config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	cfg.ServerURL = strings.TrimRight(cfg.ServerURL, "/")
	return &cfg, nil
}

func saveConfig(path string, cfg *config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func ensureRegistered(client *http.Client, configPath string, cfg *config) error {
	if cfg.AgentID != "" && cfg.AgentToken != "" {
		return nil
	}

	info, err := collectHostInfo()
	if err != nil {
		return err
	}
	payload := map[string]any{
		"enrollmentToken": cfg.EnrollmentToken,
		"hostname":        info.Hostname,
		"ip":              info.IP,
		"os":              info.OS,
		"kernel":          info.Kernel,
		"arch":            info.Arch,
		"version":         version,
	}

	var result registerResponse
	if err := postJSON(client, cfg.ServerURL+"/api/v1/public/agents/register", payload, &result); err != nil {
		return fmt.Errorf("Agent注册失败: %w", err)
	}
	if result.AgentID == "" || result.AgentToken == "" {
		return fmt.Errorf("Agent注册响应缺少认证信息")
	}

	cfg.AgentID = result.AgentID
	cfg.AgentToken = result.AgentToken
	cfg.EnrollmentToken = ""
	if result.Interval > 0 && cfg.Interval <= 0 {
		cfg.Interval = result.Interval
	}
	if err := saveConfig(configPath, cfg); err != nil {
		return err
	}

	logf("Agent注册成功: %s", cfg.AgentID)
	return nil
}

func collectAndReport(client *http.Client, cfg *config) error {
	info, err := collectHostInfo()
	if err != nil {
		return err
	}
	payload := metricsPayload{
		AgentID:     cfg.AgentID,
		AgentToken:  cfg.AgentToken,
		Hostname:    info.Hostname,
		IP:          info.IP,
		Version:     version,
		OS:          info.OS,
		Kernel:      info.Kernel,
		Arch:        info.Arch,
		CPUCores:    info.CPUCores,
		CPUUsage:    info.CPUUsage,
		MemoryTotal: info.MemoryTotal,
		MemoryUsed:  info.MemoryUsed,
		MemoryUsage: info.MemoryUsage,
		DiskTotal:   info.DiskTotal,
		DiskUsed:    info.DiskUsed,
		DiskUsage:   info.DiskUsage,
		Uptime:      info.Uptime,
	}
	if err := postJSON(client, cfg.ServerURL+"/api/v1/public/agents/metrics", payload, nil); err != nil {
		return err
	}
	logf("上报成功 CPU %.1f%% 内存 %.1f%% 磁盘 %.1f%%", info.CPUUsage, info.MemoryUsage, info.DiskUsage)
	return nil
}

func postJSON(client *http.Client, target string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var wrapped apiResponse
	if err := json.Unmarshal(respBody, &wrapped); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	if wrapped.Code != 0 && wrapped.Code != 200 {
		return errors.New(wrapped.Message)
	}
	if out != nil && len(wrapped.Data) > 0 {
		if err := json.Unmarshal(wrapped.Data, out); err != nil {
			return fmt.Errorf("解析响应数据失败: %w", err)
		}
	}
	return nil
}

type hostInfo struct {
	Hostname    string
	IP          string
	OS          string
	Kernel      string
	Arch        string
	CPUCores    int
	CPUUsage    float64
	MemoryTotal uint64
	MemoryUsed  uint64
	MemoryUsage float64
	DiskTotal   uint64
	DiskUsed    uint64
	DiskUsage   float64
	Uptime      string
}

func collectHostInfo() (*hostInfo, error) {
	hostname, _ := os.Hostname()
	osName := readOSRelease()
	kernel := readCommandlessKernel()
	arch := runtime.GOARCH
	cpuUsage := collectCPUUsage()
	memTotal, memUsed, memUsage := collectMemory()
	diskTotal, diskUsed, diskUsage := collectDisk()
	return &hostInfo{
		Hostname:    hostname,
		IP:          discoverIP(),
		OS:          osName,
		Kernel:      kernel,
		Arch:        arch,
		CPUCores:    runtime.NumCPU(),
		CPUUsage:    cpuUsage,
		MemoryTotal: memTotal,
		MemoryUsed:  memUsed,
		MemoryUsage: memUsage,
		DiskTotal:   diskTotal,
		DiskUsed:    diskUsed,
		DiskUsage:   diskUsage,
		Uptime:      collectUptime(),
	}, nil
}

func discoverIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			return addr.IP.String()
		}
	}
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip4 := ip.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return ""
}

func readOSRelease() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return runtime.GOOS
	}
	fields := parseKeyValueFile(string(data))
	if v := fields["PRETTY_NAME"]; v != "" {
		return v
	}
	if v := fields["NAME"]; v != "" {
		return v
	}
	return runtime.GOOS
}

func readCommandlessKernel() string {
	var uts syscall.Utsname
	if err := syscall.Uname(&uts); err != nil {
		return ""
	}
	return int8ArrayToString(uts.Release[:])
}

func int8ArrayToString(values []int8) string {
	var b strings.Builder
	for _, v := range values {
		if v == 0 {
			break
		}
		b.WriteByte(byte(v))
	}
	return b.String()
}

func parseKeyValueFile(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		result[parts[0]] = strings.Trim(parts[1], `"`)
	}
	return result
}

func collectCPUUsage() float64 {
	total1, idle1, err := readCPUStat()
	if err != nil {
		return 0
	}
	time.Sleep(time.Second)
	total2, idle2, err := readCPUStat()
	if err != nil {
		return 0
	}
	totalDelta := total2 - total1
	idleDelta := idle2 - idle1
	if totalDelta <= 0 {
		return 0
	}
	return round2((1 - float64(idleDelta)/float64(totalDelta)) * 100)
}

func readCPUStat() (uint64, uint64, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		var total uint64
		for _, raw := range fields[1:] {
			v, _ := strconv.ParseUint(raw, 10, 64)
			total += v
		}
		idle, _ := strconv.ParseUint(fields[4], 10, 64)
		return total, idle, nil
	}
	return 0, 0, fmt.Errorf("未找到CPU统计")
}

func collectMemory() (uint64, uint64, float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0
	}
	values := make(map[string]uint64)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		v, _ := strconv.ParseUint(fields[1], 10, 64)
		values[strings.TrimSuffix(fields[0], ":")] = v * 1024
	}
	total := values["MemTotal"]
	available := values["MemAvailable"]
	used := uint64(0)
	if total > available {
		used = total - available
	}
	usage := 0.0
	if total > 0 {
		usage = round2(float64(used) / float64(total) * 100)
	}
	return total, used, usage
}

func collectDisk() (uint64, uint64, float64) {
	mounts := readMountPoints()
	if len(mounts) == 0 {
		mounts = []string{"/"}
	}
	var total, free uint64
	seen := make(map[string]bool)
	for _, mount := range mounts {
		var stat syscall.Statfs_t
		if err := syscall.Statfs(mount, &stat); err != nil {
			continue
		}
		key := fmt.Sprintf("%d:%d", stat.Fsid.X__val[0], stat.Fsid.X__val[1])
		if seen[key] {
			continue
		}
		seen[key] = true
		total += stat.Blocks * uint64(stat.Bsize)
		free += stat.Bavail * uint64(stat.Bsize)
	}
	used := uint64(0)
	if total > free {
		used = total - free
	}
	usage := 0.0
	if total > 0 {
		usage = round2(float64(used) / float64(total) * 100)
	}
	return total, used, usage
}

func readMountPoints() []string {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return nil
	}
	excluded := map[string]bool{
		"proc": true, "sysfs": true, "tmpfs": true, "devtmpfs": true,
		"devpts": true, "cgroup": true, "cgroup2": true, "overlay": false,
		"squashfs": true, "securityfs": true, "pstore": true, "debugfs": true,
		"tracefs": true, "fusectl": true, "configfs": true, "autofs": true,
	}
	var mounts []string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		fsType := fields[2]
		if excluded[fsType] {
			continue
		}
		mounts = append(mounts, fields[1])
	}
	return mounts
}

func collectUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return ""
	}
	secondsFloat, _ := strconv.ParseFloat(fields[0], 64)
	seconds := int64(secondsFloat)
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func logf(format string, args ...any) {
	fmt.Printf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "opshub-agent: %v\n", err)
	os.Exit(1)
}
