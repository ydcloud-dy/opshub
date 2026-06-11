package asset

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const agentBinaryDir = "data/agent-binaries"

var agentBinaryBuildMu sync.Mutex

func validateAgentBinaryFilename(filename string) error {
	switch filename {
	case "opshub-agent-linux-amd64", "opshub-agent-linux-arm64":
		return nil
	default:
		return fmt.Errorf("unsupported agent binary %q", filename)
	}
}

func ensureAgentBinary(filename string) (string, error) {
	if err := validateAgentBinaryFilename(filename); err != nil {
		return "", err
	}

	if binaryPath, ok := findExistingAgentBinary(filename); ok {
		return binaryPath, nil
	}

	agentBinaryBuildMu.Lock()
	defer agentBinaryBuildMu.Unlock()

	if binaryPath, ok := findExistingAgentBinary(filename); ok {
		return binaryPath, nil
	}

	arch := strings.TrimPrefix(filename, "opshub-agent-linux-")
	sourceRoot, err := findAgentSourceRoot()
	if err != nil {
		return "", err
	}

	outDir := filepath.Join(sourceRoot, agentBinaryDir)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("创建Agent二进制目录失败: %w", err)
	}
	goCacheDir := filepath.Join(sourceRoot, ".gocache")
	if err := os.MkdirAll(goCacheDir, 0755); err != nil {
		return "", fmt.Errorf("创建Go构建缓存目录失败: %w", err)
	}

	outPath := filepath.Join(outDir, filename)
	cmd := exec.Command("go", "build", "-ldflags", "-X main.version=0.1.0", "-o", outPath, "./cmd/opshub-agent")
	cmd.Dir = sourceRoot
	cmd.Env = withBuildEnv(os.Environ(), map[string]string{
		"GOOS":   "linux",
		"GOARCH": arch,
		"GOCACHE": goCacheDir,
	})

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("自动构建Agent二进制失败，请确认OpsHub服务器已安装Go且保留完整源码: %w\n%s", err, strings.TrimSpace(output.String()))
	}

	if err := os.Chmod(outPath, 0755); err != nil {
		return "", fmt.Errorf("设置Agent二进制权限失败: %w", err)
	}
	return outPath, nil
}

func ensureLinuxAgentBinaries() error {
	for _, filename := range []string{"opshub-agent-linux-amd64", "opshub-agent-linux-arm64"} {
		if _, err := ensureAgentBinary(filename); err != nil {
			return err
		}
	}
	return nil
}

func validateAgentServerBinaries(serverURL string) error {
	serverURL = strings.TrimRight(strings.TrimSpace(serverURL), "/")
	parsed, err := url.Parse(serverURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("Agent访问地址无效，请填写 http:// 或 https:// 开头的 OpsHub/Agent Gateway 地址")
	}

	client := &http.Client{Timeout: 12 * time.Second}
	for _, filename := range []string{"opshub-agent-linux-amd64", "opshub-agent-linux-arm64"} {
		binaryURL := serverURL + "/api/v1/public/agents/binaries/" + filename
		if err := probeAgentBinaryURL(client, binaryURL); err != nil {
			return err
		}
	}
	return nil
}

func probeAgentBinaryURL(client *http.Client, binaryURL string) error {
	req, err := http.NewRequest(http.MethodGet, binaryURL, nil)
	if err != nil {
		return fmt.Errorf("Agent二进制地址无效: %w", err)
	}
	req.Header.Set("Range", "bytes=0-0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Agent访问地址不可达，请确认目标主机也能访问该地址: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	detail := strings.TrimSpace(string(body))
	if detail != "" {
		return fmt.Errorf("Agent访问地址不可用，下载二进制返回 HTTP %d: %s", resp.StatusCode, detail)
	}
	return fmt.Errorf("Agent访问地址不可用，下载二进制返回 HTTP %d", resp.StatusCode)
}

func findExistingAgentBinary(filename string) (string, bool) {
	for _, baseDir := range agentBinaryBaseDirs() {
		candidate := filepath.Join(baseDir, agentBinaryDir, filename)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Size() > 0 {
			return candidate, true
		}
	}
	return "", false
}

func agentBinaryBaseDirs() []string {
	seen := map[string]bool{}
	var dirs []string
	add := func(dir string) {
		if dir == "" {
			return
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			abs = dir
		}
		if !seen[abs] {
			seen[abs] = true
			dirs = append(dirs, abs)
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		add(cwd)
	}
	if exe, err := os.Executable(); err == nil {
		add(filepath.Dir(exe))
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		add(filepath.Clean(filepath.Join(filepath.Dir(file), "../../..")))
	}
	return dirs
}

func findAgentSourceRoot() (string, error) {
	for _, baseDir := range agentBinaryBaseDirs() {
		if root, ok := walkUpForAgentSource(baseDir); ok {
			return root, nil
		}
	}
	return "", fmt.Errorf("未找到OpsHub源码目录，无法自动构建Agent二进制")
}

func walkUpForAgentSource(start string) (string, bool) {
	current := filepath.Clean(start)
	for i := 0; i < 8; i++ {
		goMod := filepath.Join(current, "go.mod")
		agentMain := filepath.Join(current, "cmd", "opshub-agent", "main.go")
		if fileExists(goMod) && fileExists(agentMain) {
			return current, true
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func withBuildEnv(env []string, overrides map[string]string) []string {
	result := make([]string, 0, len(env)+len(overrides))
	for _, item := range env {
		key, _, ok := strings.Cut(item, "=")
		if !ok || overrides[key] != "" {
			continue
		}
		result = append(result, item)
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
}
