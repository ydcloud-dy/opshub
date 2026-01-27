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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/ydcloud-dy/opshub/plugins/kubernetes/service"
)

// ArthasHandler Arthas诊断处理器
type ArthasHandler struct {
	clusterService *service.ClusterService
	db             *gorm.DB
}

// NewArthasHandler 创建Arthas处理器
func NewArthasHandler(clusterService *service.ClusterService, db *gorm.DB) *ArthasHandler {
	return &ArthasHandler{
		clusterService: clusterService,
		db:             db,
	}
}

// ArthasCommandRequest Arthas命令请求
type ArthasCommandRequest struct {
	ClusterID uint   `json:"clusterId" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Pod       string `json:"pod" binding:"required"`
	Container string `json:"container" binding:"required"`
	Command   string `json:"command" binding:"required"`
	ProcessID string `json:"processId"` // Java进程ID，如果为空则自动检测
}

// ProcessInfo Java进程信息
type ProcessInfo struct {
	PID         string `json:"pid"`
	MainClass   string `json:"mainClass"`
	CommandLine string `json:"commandLine"`
}

// ThreadInfo 线程信息
type ThreadInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Group       string `json:"group"`
	Priority    string `json:"priority"`
	State       string `json:"state"`
	CPU         string `json:"cpu"`
	DeltaTime   string `json:"deltaTime"`
	Time        string `json:"time"`
	Interrupted bool   `json:"interrupted"`
	Daemon      bool   `json:"daemon"`
}

// MemoryInfo 内存信息
type MemoryInfo struct {
	Type  string `json:"type"`
	Used  string `json:"used"`
	Total string `json:"total"`
	Max   string `json:"max"`
	Usage string `json:"usage"`
}

// GCInfo GC信息
type GCInfo struct {
	Name            string `json:"name"`
	CollectionCount int64  `json:"collectionCount"`
	CollectionTime  int64  `json:"collectionTime"`
}

// ArthasRuntimeInfo Arthas运行时信息
type ArthasRuntimeInfo struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// DashboardData 控制面板数据
type DashboardData struct {
	Threads   []ThreadInfo        `json:"threads"`
	Memory    []MemoryInfo        `json:"memory"`
	GC        []GCInfo            `json:"gc"`
	Runtime   []ArthasRuntimeInfo `json:"runtime"`
	RawOutput string              `json:"rawOutput"`
}

// ListJavaProcesses 列出Pod中的Java进程
func (h *ArthasHandler) ListJavaProcesses(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Query("namespace")
	pod := c.Query("pod")
	container := c.Query("container")

	if clusterIDStr == "" || namespace == "" || pod == "" || container == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要参数",
		})
		return
	}

	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 执行 jps -l 命令获取Java进程列表
	// 如果jps不存在，则尝试用ps命令（使用 -o pid,args 格式，输出 PID 和命令行）
	output, err := h.execCommand(c.Request.Context(), uint(clusterID), currentUserID.(uint), namespace, pod, container, []string{"sh", "-c", "jps -l 2>/dev/null || ps -eo pid,args 2>/dev/null | grep java | grep -v grep || echo ''"})

	// 如果命令执行失败（例如容器中没有sh），返回空数组而不是错误
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data":    []ProcessInfo{},
		})
		return
	}

	// 解析进程列表
	processes := parseJavaProcesses(output)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    processes,
	})
}

// parseJavaProcesses 解析Java进程列表
func parseJavaProcesses(output string) []ProcessInfo {
	var processes []ProcessInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// jps -l 格式: PID MainClass
		// ps -eo pid,args 格式: PID command args...
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			pid := parts[0]

			// 验证第一个字段是否是数字（PID）
			if _, err := strconv.Atoi(pid); err != nil {
				// 如果不是数字，可能是 ps aux 格式，尝试解析第二个字段作为 PID
				if len(parts) >= 2 {
					if _, err := strconv.Atoi(parts[1]); err == nil {
						pid = parts[1]
						// 跳过用户名和PID，剩余部分作为命令
						if len(parts) > 2 {
							parts = append([]string{pid}, parts[2:]...)
						}
					} else {
						continue // 无法解析 PID，跳过此行
					}
				}
			}

			// 跳过 jps 自身
			mainClass := ""
			if len(parts) >= 2 {
				mainClass = parts[1]
			}
			if strings.Contains(mainClass, "Jps") || strings.Contains(mainClass, "sun.tools.jps") {
				continue
			}

			// 跳过 arthas-boot.jar 进程（这些是诊断工具本身）
			cmdLine := strings.Join(parts[1:], " ")
			if strings.Contains(cmdLine, "arthas-boot.jar") {
				continue
			}

			processes = append(processes, ProcessInfo{
				PID:       pid,
				MainClass: mainClass,
			})
		}
	}

	return processes
}

// ExecuteArthasCommand 执行Arthas命令（一次性命令）
func (h *ArthasHandler) ExecuteArthasCommand(c *gin.Context) {
	var req ArthasCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 构建Arthas命令
	arthasCmd := h.buildArthasCommand(req.ProcessID, req.Command)

	// 执行命令
	output, err := h.execCommand(c.Request.Context(), req.ClusterID, currentUserID.(uint), req.Namespace, req.Pod, req.Container, arthasCmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "执行Arthas命令失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    output,
	})
}

// buildArthasCommand 构建Arthas命令
func (h *ArthasHandler) buildArthasCommand(processID string, command string) []string {
	// 使用 arthas-boot.jar 执行命令
	script := fmt.Sprintf(`
# 下载 arthas-boot.jar 如果不存在
if [ ! -f /tmp/arthas-boot.jar ]; then
    echo "[INFO] Downloading arthas-boot.jar..."
    curl -sL -o /tmp/arthas-boot.jar https://arthas.aliyun.com/arthas-boot.jar 2>/dev/null || \
    wget -q -O /tmp/arthas-boot.jar https://arthas.aliyun.com/arthas-boot.jar 2>/dev/null
    if [ ! -f /tmp/arthas-boot.jar ]; then
        echo "[ERROR] Failed to download arthas-boot.jar"
        exit 1
    fi
fi

TARGET_PID=%s
COMMAND='%s'
echo "[INFO] Executing Arthas command on process $TARGET_PID: $COMMAND"

# 查找可用的 JDK（需要有 tools.jar 或 jdk.attach 模块）
find_jdk() {
    # 常见的 Java 安装路径
    JAVA_PATHS="/usr/lib/jvm /opt/java /opt/jdk /usr/java /opt /Library/Java/JavaVirtualMachines"

    for base in $JAVA_PATHS; do
        if [ -d "$base" ]; then
            # 查找所有 java 可执行文件
            for java_bin in $(find "$base" -name "java" -type f 2>/dev/null | grep -E "/bin/java$" | head -10); do
                java_home=$(dirname $(dirname "$java_bin"))

                # 检查是否有 tools.jar (Java 8) 或 jmods 目录 (Java 9+)
                if [ -f "$java_home/lib/tools.jar" ]; then
                    echo "[DEBUG] Found JDK with tools.jar: $java_home" >&2
                    echo "$java_bin"
                    return 0
                fi
                if [ -d "$java_home/jmods" ]; then
                    echo "[DEBUG] Found JDK with jmods: $java_home" >&2
                    echo "$java_bin"
                    return 0
                fi
            done
        fi
    done

    # 如果没找到，尝试使用 JAVA_HOME
    if [ -n "$JAVA_HOME" ]; then
        if [ -f "$JAVA_HOME/lib/tools.jar" ] || [ -d "$JAVA_HOME/jmods" ]; then
            echo "$JAVA_HOME/bin/java"
            return 0
        fi
    fi

    # 没有找到 JDK
    return 1
}

# 查找 JDK
JAVA_BIN=$(find_jdk)
if [ -z "$JAVA_BIN" ]; then
    echo "[ERROR] 未找到 JDK 环境"
    echo "[ERROR] Arthas 需要完整的 JDK（不是 JRE）才能运行"
    echo "[ERROR] 当前 Java 环境: $(java -version 2>&1 | head -1)"
    echo ""
    echo "[HINT] 解决方案:"
    echo "  1. 修改容器基础镜像，使用 JDK 而不是 JRE"
    echo "     例如: eclipse-temurin:17-jdk 或 openjdk:17-jdk"
    echo "  2. 在 Dockerfile 中安装完整的 JDK"
    exit 1
fi

echo "[INFO] Using Java: $JAVA_BIN"

# 使用固定端口，基于进程ID计算
BASE_PORT=$((3658 + (TARGET_PID %% 100)))
echo "[INFO] Using telnet port: $BASE_PORT"

# 等待端口就绪的函数
wait_for_port() {
    local port=$1
    local max_wait=15
    local waited=0
    while [ $waited -lt $max_wait ]; do
        # 检查端口是否在监听
        if (echo > /dev/tcp/127.0.0.1/$port) 2>/dev/null; then
            echo "[INFO] Port $port is ready"
            return 0
        fi
        sleep 1
        waited=$((waited + 1))
        echo "[INFO] Waiting for port $port... ($waited/$max_wait)"
    done
    return 1
}

# 主执行逻辑
PORT=$BASE_PORT
MAX_ATTEMPTS=3
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    ATTEMPT=$((ATTEMPT + 1))
    echo "[INFO] Attempt $ATTEMPT/$MAX_ATTEMPTS on port $PORT"

    # 执行命令，使用找到的 JDK
    OUTPUT=$($JAVA_BIN -jar /tmp/arthas-boot.jar $TARGET_PID --telnet-port $PORT --http-port -1 -c "$COMMAND" 2>&1)

    # 检查是否成功获取到数据
    if echo "$OUTPUT" | grep -qE "ID.*NAME|heap|nonheap|Memory|THREAD|RUNNABLE|WAITING|BLOCKED|TIMED_|@|gc\.|eden|os\.|java\.|user\.|sun\."; then
        echo "$OUTPUT"
        exit 0
    fi

    # 检查错误类型
    if echo "$OUTPUT" | grep -q "telnet port.*is used"; then
        echo "[WARN] Port $PORT already in use, trying next port..."
        PORT=$((PORT + 1))
        sleep 1
        continue
    fi

    # 如果 attach 成功但连接失败，等待 telnet 服务启动
    if echo "$OUTPUT" | grep -q "Attach process.*success"; then
        echo "[INFO] Agent attached, waiting for telnet server to start..."

        # 等待端口就绪
        if wait_for_port $PORT; then
            # 端口就绪后重新执行命令
            echo "[INFO] Retrying command..."
            OUTPUT=$($JAVA_BIN -jar /tmp/arthas-boot.jar $TARGET_PID --telnet-port $PORT --http-port -1 -c "$COMMAND" 2>&1)
            if echo "$OUTPUT" | grep -qE "ID.*NAME|heap|nonheap|Memory|THREAD|RUNNABLE|WAITING|BLOCKED|TIMED_|@|gc\.|eden|os\.|java\.|user\.|sun\."; then
                echo "$OUTPUT"
                exit 0
            fi
        fi

        # 如果还是失败，显示输出并继续尝试
        echo "$OUTPUT"
    else
        echo "$OUTPUT"
    fi

    # 等待一段时间后重试
    if [ $ATTEMPT -lt $MAX_ATTEMPTS ]; then
        echo "[INFO] Waiting before retry..."
        sleep 5
    fi
done

# 最后一次尝试用不同端口
FINAL_PORT=$((BASE_PORT + 100))
echo "[INFO] Final attempt with port $FINAL_PORT"
OUTPUT=$($JAVA_BIN -jar /tmp/arthas-boot.jar $TARGET_PID --telnet-port $FINAL_PORT --http-port -1 -c "$COMMAND" 2>&1)
echo "$OUTPUT"

if echo "$OUTPUT" | grep -qE "ID.*NAME|heap|nonheap|Memory|THREAD|RUNNABLE|WAITING|BLOCKED|TIMED_|@|gc\.|eden|os\.|java\.|user\.|sun\."; then
    exit 0
fi

echo "[ERROR] Failed to execute Arthas command after multiple attempts"
exit 1
`, processID, command)

	return []string{"sh", "-c", script}
}

// buildArthasCommandForOgnl 构建用于 ognl 命令的 Arthas 脚本（处理特殊字符）
func (h *ArthasHandler) buildArthasCommandForOgnl(processID string, ognlExpr string) []string {
	// 使用 arthas-boot.jar 执行 ognl 命令
	script := fmt.Sprintf(`
# 下载 arthas-boot.jar 如果不存在
if [ ! -f /tmp/arthas-boot.jar ]; then
    echo "[INFO] Downloading arthas-boot.jar..."
    curl -sL -o /tmp/arthas-boot.jar https://arthas.aliyun.com/arthas-boot.jar 2>/dev/null || \
    wget -q -O /tmp/arthas-boot.jar https://arthas.aliyun.com/arthas-boot.jar 2>/dev/null
    if [ ! -f /tmp/arthas-boot.jar ]; then
        echo "[ERROR] Failed to download arthas-boot.jar"
        exit 1
    fi
fi

TARGET_PID=%s
echo "[INFO] Executing Arthas ognl command on process $TARGET_PID"

# 查找可用的 JDK
find_jdk() {
    JAVA_PATHS="/usr/lib/jvm /opt/java /opt/jdk /usr/java /Library/Java/JavaVirtualMachines"
    for base in $JAVA_PATHS; do
        if [ -d "$base" ]; then
            for java_bin in $(find "$base" -name "java" -type f 2>/dev/null | grep -E "/bin/java$"); do
                java_home=$(dirname $(dirname "$java_bin"))
                if [ -f "$java_home/lib/tools.jar" ] || [ -d "$java_home/jmods" ]; then
                    echo "$java_bin"
                    return 0
                fi
            done
        fi
    done
    if [ -n "$JAVA_HOME" ]; then
        if [ -f "$JAVA_HOME/lib/tools.jar" ] || [ -d "$JAVA_HOME/jmods" ]; then
            echo "$JAVA_HOME/bin/java"
            return 0
        fi
    fi
    echo "java"
    return 1
}

JAVA_BIN=$(find_jdk)
echo "[INFO] Using Java: $JAVA_BIN"

# 使用固定端口，基于进程ID计算
BASE_PORT=$((3658 + (TARGET_PID %% 100)))

# 等待端口就绪的函数
wait_for_port() {
    local port=$1
    local max_wait=15
    local waited=0
    while [ $waited -lt $max_wait ]; do
        if (echo > /dev/tcp/127.0.0.1/$port) 2>/dev/null; then
            return 0
        fi
        sleep 1
        waited=$((waited + 1))
    done
    return 1
}

# 主执行逻辑
PORT=$BASE_PORT
MAX_ATTEMPTS=3
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    ATTEMPT=$((ATTEMPT + 1))
    echo "[INFO] Attempt $ATTEMPT/$MAX_ATTEMPTS on port $PORT"

    OUTPUT=$($JAVA_BIN -jar /tmp/arthas-boot.jar $TARGET_PID --telnet-port $PORT --http-port -1 -c "ognl '%s'" 2>&1)

    # 检查是否成功
    if echo "$OUTPUT" | grep -qE "@HashMap|@Properties|@String|@Integer|@Long"; then
        echo "$OUTPUT"
        exit 0
    fi

    # 端口冲突
    if echo "$OUTPUT" | grep -q "telnet port.*is used"; then
        PORT=$((PORT + 1))
        sleep 1
        continue
    fi

    # attach 成功但连接失败
    if echo "$OUTPUT" | grep -q "Attach process.*success"; then
        echo "[INFO] Agent attached, waiting for telnet server..."
        if wait_for_port $PORT; then
            OUTPUT=$($JAVA_BIN -jar /tmp/arthas-boot.jar $TARGET_PID --telnet-port $PORT --http-port -1 -c "ognl '%s'" 2>&1)
            if echo "$OUTPUT" | grep -qE "@HashMap|@Properties|@String|@Integer|@Long"; then
                echo "$OUTPUT"
                exit 0
            fi
        fi
        echo "$OUTPUT"
    elif echo "$OUTPUT" | grep -qE "No process|Can not find"; then
        echo "[ERROR] Process $TARGET_PID not found"
        exit 1
    elif echo "$OUTPUT" | grep -qE "Unable to attach|attach not supported|tools.jar"; then
        echo "[ERROR] This JVM does not support Arthas attach (JDK required, not JRE)"
        echo "$OUTPUT"
        exit 1
    else
        echo "$OUTPUT"
    fi

    if [ $ATTEMPT -lt $MAX_ATTEMPTS ]; then
        sleep 5
    fi
done

echo "[ERROR] Failed to execute Arthas ognl command"
exit 1
`, processID, ognlExpr, ognlExpr)

	return []string{"sh", "-c", script}
}

// GetDashboard 获取控制面板信息
func (h *ArthasHandler) GetDashboard(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Query("namespace")
	pod := c.Query("pod")
	container := c.Query("container")
	processID := c.Query("processId")

	if clusterIDStr == "" || namespace == "" || pod == "" || container == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要参数",
		})
		return
	}

	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 构建Arthas命令
	arthasCmd := h.buildArthasCommand(processID, "dashboard -n 1")

	// 执行命令（带超时，Arthas 首次启动可能需要较长时间）
	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	output, err := h.execCommand(ctx, uint(clusterID), currentUserID.(uint), namespace, pod, container, arthasCmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "执行Arthas命令失败: " + err.Error(),
		})
		return
	}

	// 解析dashboard输出
	dashboardData := parseDashboardOutput(output)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    dashboardData,
	})
}

// parseDashboardOutput 解析Arthas dashboard输出
func parseDashboardOutput(output string) DashboardData {
	data := DashboardData{
		Threads:   []ThreadInfo{},
		Memory:    []MemoryInfo{},
		GC:        []GCInfo{},
		Runtime:   []ArthasRuntimeInfo{},
		RawOutput: output,
	}

	// 移除ANSI颜色代码
	output = stripANSI(output)

	lines := strings.Split(output, "\n")
	section := "" // 当前解析的区域: threads, memory, runtime

	for _, line := range lines {
		// 移除回车符
		line = strings.TrimRight(line, "\r")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 跳过arthas启动信息和banner
		if strings.HasPrefix(line, "[INFO]") || strings.HasPrefix(line, "[arthas@") ||
			strings.HasPrefix(line, "[WARN]") || strings.HasPrefix(line, "[DEBUG]") ||
			strings.HasPrefix(line, "[ERROR]") ||
			strings.Contains(line, "wiki") || strings.Contains(line, "tutorials") ||
			strings.Contains(line, "version") || strings.Contains(line, "main_class") ||
			strings.Contains(line, "start_time") || strings.Contains(line, "current_time") ||
			strings.HasPrefix(line, "pid") || strings.HasPrefix(line, "time") ||
			strings.Contains(line, "Process ends") || strings.Contains(line, "ARTHAS") ||
			strings.Contains(line, "arthas-client") || strings.Contains(line, "arthas-boot") ||
			strings.Contains(line, "Attach process") || strings.Contains(line, "JAVA_HOME") ||
			strings.Contains(line, "arthas home") || strings.Contains(line, "Download") {
			continue
		}

		// 跳过 ASCII 艺术 banner 行（包含大量特殊字符的行）
		if isAsciiArtLine(line) {
			continue
		}

		// 检测区域标题
		if strings.HasPrefix(line, "ID") && strings.Contains(line, "NAME") {
			section = "threads"
			continue
		}
		if strings.HasPrefix(line, "Memory") {
			section = "memory"
			continue
		}
		if strings.HasPrefix(line, "Runtime") {
			section = "runtime"
			continue
		}

		// 根据区域解析数据
		switch section {
		case "threads":
			// 如果遇到Memory行，切换到memory区域
			if strings.HasPrefix(line, "Memory") {
				section = "memory"
				continue
			}
			thread := parseThreadLineV2(line)
			if thread.ID != "" && thread.ID != "ID" && thread.ID != "-1" {
				data.Threads = append(data.Threads, thread)
			}
			// 限制线程数量为TOP 10
			if len(data.Threads) >= 10 {
				section = "" // 停止收集线程
			}
		case "memory":
			// 如果遇到Runtime行，切换到runtime区域
			if strings.HasPrefix(line, "Runtime") {
				section = "runtime"
				continue
			}
			// 解析内存和GC信息（它们在同一行）
			parseMemoryAndGCLine(line, &data)
		case "runtime":
			// Runtime 行格式: os.name    Linux
			parseRuntimeLine(line, &data)
		}
	}

	return data
}

// isAsciiArtLine 检测是否是 ASCII 艺术行（Arthas banner）
func isAsciiArtLine(line string) bool {
	// ASCII 艺术行通常包含大量特殊字符如 | \ / - _ ` ' 等
	specialChars := 0
	alphaNumeric := 0

	for _, c := range line {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			alphaNumeric++
		} else if c == '|' || c == '\\' || c == '/' || c == '-' || c == '_' ||
			c == '`' || c == '\'' || c == ',' || c == '.' || c == '*' ||
			c == '[' || c == ']' || c == '(' || c == ')' {
			specialChars++
		}
	}

	// 如果特殊字符比字母数字多，认为是 ASCII 艺术行
	if len(line) > 10 && specialChars > alphaNumeric {
		return true
	}

	// 检查常见的 ASCII 艺术模式
	if strings.Contains(line, "---") && strings.Contains(line, "\\") {
		return true
	}
	if strings.Contains(line, ",-") && strings.Contains(line, "-.") {
		return true
	}
	if strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") && len(line) > 20 {
		return true
	}

	return false
}

// stripANSI 移除ANSI转义码
func stripANSI(str string) string {
	// 匹配ANSI转义序列: \x1b[...m 或 \033[...m
	result := strings.Builder{}
	i := 0
	for i < len(str) {
		if i+1 < len(str) && (str[i] == '\x1b' || str[i] == '\033') && str[i+1] == '[' {
			// 跳过直到遇到'm'
			j := i + 2
			for j < len(str) && str[j] != 'm' {
				j++
			}
			i = j + 1
			continue
		}
		result.WriteByte(str[i])
		i++
	}
	return result.String()
}

// parseThreadLineV2 解析线程行（新格式）
func parseThreadLineV2(line string) ThreadInfo {
	// 线程行格式: ID NAME GROUP PRIORI STATE %CPU DELTA_ TIME INTER DAEMON
	// 例如: 14 com.alibaba.nacos.c main 5 TIMED_ 0.0 0.000 22:54. false true
	parts := strings.Fields(line)
	if len(parts) < 8 {
		return ThreadInfo{}
	}

	// 第一个字段是ID
	id := parts[0]
	// 验证ID是数字
	if _, err := strconv.Atoi(id); err != nil {
		return ThreadInfo{}
	}

	// 从后往前解析固定字段
	daemon := false
	interrupted := false
	time := ""
	deltaTime := ""
	cpu := ""
	state := ""
	priority := ""
	group := ""
	name := ""

	n := len(parts)
	if n >= 2 {
		daemon = parts[n-1] == "true"
	}
	if n >= 3 {
		interrupted = parts[n-2] == "true"
	}
	if n >= 4 {
		time = parts[n-3]
	}
	if n >= 5 {
		deltaTime = parts[n-4]
	}
	if n >= 6 {
		cpu = parts[n-5]
	}
	if n >= 7 {
		state = parts[n-6]
		// 补全被截断的状态
		if state == "TIMED_" {
			state = "TIMED_WAITING"
		} else if state == "WAITIN" {
			state = "WAITING"
		} else if state == "RUNNAB" {
			state = "RUNNABLE"
		} else if state == "BLOCKE" {
			state = "BLOCKED"
		}
	}
	if n >= 8 {
		priority = parts[n-7]
	}
	if n >= 9 {
		group = parts[n-8]
	}
	// 名称是ID和group之间的所有内容
	if n >= 10 {
		nameEndIdx := n - 8
		name = strings.Join(parts[1:nameEndIdx], " ")
	} else if n >= 9 {
		name = parts[1]
	}

	return ThreadInfo{
		ID:          id,
		Name:        name,
		Group:       group,
		Priority:    priority,
		State:       state,
		CPU:         cpu,
		DeltaTime:   deltaTime,
		Time:        time,
		Interrupted: interrupted,
		Daemon:      daemon,
	}
}

// parseMemoryAndGCLine 解析内存和GC行
func parseMemoryAndGCLine(line string, data *DashboardData) {
	// 内存行格式可能是:
	// heap             73M   456M       1.01% gc.copy.count       637
	// eden_space       31M   126M       1.57% gc.copy.time(ms)    2653
	// nonheap          119M  124M  -1

	parts := strings.Fields(line)
	if len(parts) < 3 {
		return
	}

	// 检查是否是内存行
	memTypes := map[string]bool{
		"heap": true, "nonheap": true, "eden_space": true,
		"survivor_space": true, "tenured_gen": true, "code_cache": true,
		"metaspace": true, "ps_eden_space": true, "ps_survivor_space": true,
		"ps_old_gen": true, "g1_eden_space": true, "g1_survivor_space": true,
		"g1_old_gen": true,
	}

	firstPart := strings.ToLower(parts[0])
	if memTypes[firstPart] {
		mem := MemoryInfo{Type: parts[0]}
		if len(parts) >= 2 {
			mem.Used = parts[1]
		}
		if len(parts) >= 3 {
			mem.Total = parts[2]
		}
		// 查找使用率 (包含%)
		for i := 3; i < len(parts); i++ {
			if strings.Contains(parts[i], "%") {
				mem.Usage = parts[i]
				break
			}
			if i == 3 {
				mem.Max = parts[i]
			}
		}
		data.Memory = append(data.Memory, mem)
	}

	// 检查是否包含GC信息
	for i, part := range parts {
		if strings.HasPrefix(part, "gc.") {
			gcPart := part
			var valuePart string
			if i+1 < len(parts) {
				valuePart = parts[i+1]
			}
			parseGCInfo(gcPart, valuePart, data)
		}
	}
}

// parseGCInfo 解析GC信息
func parseGCInfo(gcPart string, valuePart string, data *DashboardData) {
	// gc.copy.count, gc.copy.time(ms), gc.marksweepcompact.count 等
	if strings.Contains(gcPart, ".count") {
		gcName := strings.Replace(gcPart, "gc.", "", 1)
		gcName = strings.Replace(gcName, ".count", "", 1)
		count, _ := strconv.ParseInt(valuePart, 10, 64)

		// 查找或创建GC条目
		found := false
		for i := range data.GC {
			if data.GC[i].Name == gcName {
				data.GC[i].CollectionCount = count
				found = true
				break
			}
		}
		if !found {
			data.GC = append(data.GC, GCInfo{Name: gcName, CollectionCount: count})
		}
	} else if strings.Contains(gcPart, ".time") {
		gcName := strings.Replace(gcPart, "gc.", "", 1)
		gcName = strings.Replace(gcName, ".time(ms)", "", 1)
		gcName = strings.Replace(gcName, ".time", "", 1)
		time, _ := strconv.ParseInt(valuePart, 10, 64)

		// 查找或创建GC条目
		found := false
		for i := range data.GC {
			if data.GC[i].Name == gcName {
				data.GC[i].CollectionTime = time
				found = true
				break
			}
		}
		if !found {
			data.GC = append(data.GC, GCInfo{Name: gcName, CollectionTime: time})
		}
	}
}

// parseRuntimeLine 解析运行时信息行
func parseRuntimeLine(line string, data *DashboardData) {
	// Runtime 行格式: os.name    Linux
	// 或者可能跨行
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		name := parts[0]
		value := strings.Join(parts[1:], " ")
		data.Runtime = append(data.Runtime, ArthasRuntimeInfo{
			Name:  name,
			Value: value,
		})
	}
}

// GetThreadList 获取线程列表
func (h *ArthasHandler) GetThreadList(c *gin.Context) {
	h.executeArthasCommandWithResponse(c, "thread")
}

// GetThreadStack 获取线程堆栈
func (h *ArthasHandler) GetThreadStack(c *gin.Context) {
	threadID := c.Query("threadId")
	if threadID == "" {
		// 获取所有线程堆栈
		h.executeArthasCommandWithResponse(c, "thread -all")
	} else {
		h.executeArthasCommandWithResponse(c, fmt.Sprintf("thread %s", threadID))
	}
}

// GetJvmInfo 获取JVM信息
func (h *ArthasHandler) GetJvmInfo(c *gin.Context) {
	h.executeArthasCommandWithResponse(c, "jvm")
}

// GetSysEnv 获取系统环境变量
func (h *ArthasHandler) GetSysEnv(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Query("namespace")
	pod := c.Query("pod")
	container := c.Query("container")
	processID := c.Query("processId")

	if clusterIDStr == "" || namespace == "" || pod == "" || container == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要参数",
		})
		return
	}

	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 使用 ognl 命令获取完整的环境变量，避免表格截断
	arthasCmd := h.buildArthasCommandForOgnl(processID, "@java.lang.System@getenv()")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	output, err := h.execCommand(ctx, uint(clusterID), currentUserID.(uint), namespace, pod, container, arthasCmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "执行Arthas命令失败: " + err.Error(),
		})
		return
	}

	// 解析 ognl 输出为键值对数组
	items := parseOgnlMapOutput(output)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    items,
	})
}

// GetSysProp 获取系统属性
func (h *ArthasHandler) GetSysProp(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Query("namespace")
	pod := c.Query("pod")
	container := c.Query("container")
	processID := c.Query("processId")

	if clusterIDStr == "" || namespace == "" || pod == "" || container == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要参数",
		})
		return
	}

	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 使用 ognl 命令获取完整的系统属性，避免表格截断
	arthasCmd := h.buildArthasCommandForOgnl(processID, "@java.lang.System@getProperties()")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	output, err := h.execCommand(ctx, uint(clusterID), currentUserID.(uint), namespace, pod, container, arthasCmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "执行Arthas命令失败: " + err.Error(),
		})
		return
	}

	// 解析 ognl 输出为键值对数组
	items := parseOgnlMapOutput(output)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    items,
	})
}

// KeyValueItem 键值对项
type KeyValueItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// parseOgnlMapOutput 解析 ognl 命令返回的 Map 输出
func parseOgnlMapOutput(output string) []KeyValueItem {
	var items []KeyValueItem

	// 移除 ANSI 颜色代码
	output = stripANSI(output)

	// ognl 输出格式类似:
	// @HashMap[
	//     @String[PATH]:@String[/usr/local/sbin:/usr/local/bin:...],
	//     @String[HOSTNAME]:@String[pod-name],
	// ]
	// 或者 Properties 格式:
	// @Properties[
	//     @String[java.version]:@String[21.0.4],
	// ]

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过空行和非数据行
		if line == "" || strings.HasPrefix(line, "[INFO]") ||
			strings.HasPrefix(line, "[arthas@") ||
			strings.HasPrefix(line, "@HashMap") ||
			strings.HasPrefix(line, "@Properties") ||
			strings.HasPrefix(line, "@UnmodifiableMap") ||
			line == "]" || line == "," ||
			strings.Contains(line, "wiki") ||
			strings.Contains(line, "tutorials") ||
			strings.Contains(line, "ARTHAS") {
			continue
		}

		// 解析 @String[key]:@String[value] 格式
		// 例如: @String[PATH]:@String[/usr/local/bin],
		if strings.Contains(line, "@String[") {
			item := parseOgnlKeyValue(line)
			if item.Key != "" {
				items = append(items, item)
			}
		}
	}

	// 按 key 排序
	sortKeyValueItems(items)

	return items
}

// parseOgnlKeyValue 解析单行 ognl 键值对
func parseOgnlKeyValue(line string) KeyValueItem {
	// 格式: @String[key]:@String[value],
	// 或者: @String[key]:@String[value]
	line = strings.TrimSuffix(line, ",")
	line = strings.TrimSpace(line)

	// 查找 ]:@ 分隔符
	sepIndex := strings.Index(line, "]:@")
	if sepIndex == -1 {
		return KeyValueItem{}
	}

	keyPart := line[:sepIndex+1]   // @String[key]
	valuePart := line[sepIndex+2:] // @String[value] 或其他类型

	// 提取 key
	key := extractStringValue(keyPart)
	if key == "" {
		return KeyValueItem{}
	}

	// 提取 value
	value := extractAnyValue(valuePart)

	return KeyValueItem{
		Key:   key,
		Value: value,
	}
}

// extractStringValue 从 @String[xxx] 格式中提取值
func extractStringValue(s string) string {
	// @String[value]
	prefix := "@String["
	if !strings.HasPrefix(s, prefix) {
		return ""
	}
	s = strings.TrimPrefix(s, prefix)
	if !strings.HasSuffix(s, "]") {
		return s
	}
	return strings.TrimSuffix(s, "]")
}

// extractAnyValue 从各种 @Type[xxx] 格式中提取值
func extractAnyValue(s string) string {
	s = strings.TrimSpace(s)

	// 处理 @String[value]
	if strings.HasPrefix(s, "@String[") {
		return extractStringValue(s)
	}

	// 处理 @Integer[123] 等数字类型
	if strings.HasPrefix(s, "@Integer[") || strings.HasPrefix(s, "@Long[") ||
		strings.HasPrefix(s, "@Double[") || strings.HasPrefix(s, "@Float[") ||
		strings.HasPrefix(s, "@Boolean[") {
		// 找到 [ 和 ] 之间的内容
		start := strings.Index(s, "[")
		end := strings.LastIndex(s, "]")
		if start != -1 && end > start {
			return s[start+1 : end]
		}
	}

	// 处理 null
	if s == "null" || s == "@null" {
		return ""
	}

	// 其他情况，尝试提取 [...] 中的内容
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start != -1 && end > start {
		return s[start+1 : end]
	}

	return s
}

// sortKeyValueItems 按 key 排序
func sortKeyValueItems(items []KeyValueItem) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].Key > items[j].Key {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

// GetPerfCounter 获取性能计数器
func (h *ArthasHandler) GetPerfCounter(c *gin.Context) {
	h.executeArthasCommandWithResponse(c, "perfcounter")
}

// GetMemory 获取内存信息
func (h *ArthasHandler) GetMemory(c *gin.Context) {
	h.executeArthasCommandWithResponse(c, "memory")
}

// DecompileClass 反编译类
func (h *ArthasHandler) DecompileClass(c *gin.Context) {
	className := c.Query("className")
	if className == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少类名参数",
		})
		return
	}
	h.executeArthasCommandWithResponse(c, fmt.Sprintf("jad %s", className))
}

// GetStaticField 获取静态字段
func (h *ArthasHandler) GetStaticField(c *gin.Context) {
	className := c.Query("className")
	fieldName := c.Query("fieldName")
	if className == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少类名参数",
		})
		return
	}

	cmd := fmt.Sprintf("getstatic %s", className)
	if fieldName != "" {
		cmd = fmt.Sprintf("getstatic %s %s", className, fieldName)
	}
	h.executeArthasCommandWithResponse(c, cmd)
}

// SearchClass 搜索类
func (h *ArthasHandler) SearchClass(c *gin.Context) {
	pattern := c.Query("pattern")
	if pattern == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少搜索模式参数",
		})
		return
	}
	h.executeArthasCommandWithResponse(c, fmt.Sprintf("sc %s", pattern))
}

// SearchMethod 搜索方法
func (h *ArthasHandler) SearchMethod(c *gin.Context) {
	className := c.Query("className")
	methodName := c.Query("methodName")
	if className == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少类名参数",
		})
		return
	}

	cmd := fmt.Sprintf("sm %s", className)
	if methodName != "" {
		cmd = fmt.Sprintf("sm %s %s", className, methodName)
	}
	h.executeArthasCommandWithResponse(c, cmd)
}

// executeArthasCommandWithResponse 执行Arthas命令并返回结果
func (h *ArthasHandler) executeArthasCommandWithResponse(c *gin.Context, command string) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Query("namespace")
	pod := c.Query("pod")
	container := c.Query("container")
	processID := c.Query("processId")

	if clusterIDStr == "" || namespace == "" || pod == "" || container == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要参数",
		})
		return
	}

	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 构建Arthas命令
	arthasCmd := h.buildArthasCommand(processID, command)

	// 执行命令（带超时，Arthas 首次启动可能需要较长时间）
	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	output, err := h.execCommand(ctx, uint(clusterID), currentUserID.(uint), namespace, pod, container, arthasCmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "执行Arthas命令失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    output,
	})
}

// execCommand 在Pod中执行命令
func (h *ArthasHandler) execCommand(ctx context.Context, clusterID, userID uint, namespace, pod, container string, command []string) (string, error) {
	restConfig, err := h.clusterService.GetRESTConfig(clusterID, userID)
	if err != nil {
		return "", fmt.Errorf("获取集群配置失败: %w", err)
	}

	serverURL, err := url.Parse(restConfig.Host)
	if err != nil {
		return "", fmt.Errorf("解析集群URL失败: %w", err)
	}

	query := url.Values{}
	query.Set("container", container)
	query.Set("stdin", "false")
	query.Set("stdout", "true")
	query.Set("stderr", "true")
	query.Set("tty", "false")

	for _, cmd := range command {
		query.Add("command", cmd)
	}

	execURL := &url.URL{
		Scheme:   serverURL.Scheme,
		Host:     serverURL.Host,
		Path:     fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/exec", namespace, pod),
		RawQuery: query.Encode(),
	}

	executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execURL)
	if err != nil {
		return "", fmt.Errorf("创建executor失败: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	// 即使命令返回非零退出码，也返回输出内容用于调试
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n--- stderr ---\n" + stderr.String()
	}

	if err != nil {
		// 如果有输出，包含在错误信息中便于调试
		if len(output) > 0 {
			return "", fmt.Errorf("执行命令失败: %s\n输出: %s", err.Error(), output)
		}
		return "", fmt.Errorf("执行命令失败: %w", err)
	}

	return output, nil
}

// ArthasWebSocket Arthas WebSocket连接（用于实时命令如trace, watch, monitor）
func (h *ArthasHandler) ArthasWebSocket(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Query("namespace")
	pod := c.Query("pod")
	container := c.Query("container")
	processID := c.Query("processId")

	if clusterIDStr == "" || namespace == "" || pod == "" || container == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要参数",
		})
		return
	}

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 升级到WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 获取REST config
	restConfig, err := h.clusterService.GetRESTConfig(uint(clusterID), currentUserID.(uint))
	if err != nil {
		h.sendWSError(conn, "获取集群配置失败: "+err.Error())
		return
	}

	serverURL, err := url.Parse(restConfig.Host)
	if err != nil {
		h.sendWSError(conn, "解析集群URL失败: "+err.Error())
		return
	}

	// 创建可取消的context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动一个goroutine来监听WebSocket命令
	commandChan := make(chan string, 10)
	stopChan := make(chan struct{})

	go func() {
		defer close(stopChan)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					// WebSocket closed normally
				}
				cancel()
				return
			}

			// 解析命令
			var cmd struct {
				Type    string `json:"type"`    // "command" or "stop"
				Command string `json:"command"` // Arthas命令
			}
			if err := json.Unmarshal(message, &cmd); err != nil {
				h.sendWSMessage(conn, "error", "无效的命令格式: "+err.Error())
				continue
			}

			if cmd.Type == "stop" {
				cancel()
				return
			}

			if cmd.Type == "command" && cmd.Command != "" {
				commandChan <- cmd.Command
			}
		}
	}()

	// 处理命令
	for {
		select {
		case <-ctx.Done():
			return
		case <-stopChan:
			return
		case command := <-commandChan:
			// 执行Arthas命令
			h.executeStreamingArthasCommand(ctx, conn, restConfig, serverURL, namespace, pod, container, processID, command)
		}
	}
}

// executeStreamingArthasCommand 执行流式Arthas命令
func (h *ArthasHandler) executeStreamingArthasCommand(ctx context.Context, conn *websocket.Conn, restConfig *rest.Config, serverURL *url.URL, namespace, pod, container, processID, command string) {
	// 这里需要实现流式输出
	// 对于 trace, watch, monitor 等命令，需要持续输出结果

	// 构建执行脚本
	script := fmt.Sprintf(`
# 下载 arthas-boot.jar 如果不存在
if [ ! -f /tmp/arthas-boot.jar ]; then
    curl -sL -o /tmp/arthas-boot.jar https://arthas.aliyun.com/arthas-boot.jar 2>/dev/null || \
    wget -q -O /tmp/arthas-boot.jar https://arthas.aliyun.com/arthas-boot.jar 2>/dev/null
fi
# 使用 arthas 执行命令
java -jar /tmp/arthas-boot.jar %s -c '%s'
`, processID, command)

	query := url.Values{}
	query.Set("container", container)
	query.Set("stdin", "true")
	query.Set("stdout", "true")
	query.Set("stderr", "true")
	query.Set("tty", "true")
	query.Add("command", "sh")
	query.Add("command", "-c")
	query.Add("command", script)

	execURL := &url.URL{
		Scheme:   serverURL.Scheme,
		Host:     serverURL.Host,
		Path:     fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/exec", namespace, pod),
		RawQuery: query.Encode(),
	}

	executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execURL)
	if err != nil {
		h.sendWSError(conn, "创建executor失败: "+err.Error())
		return
	}

	// 创建流式读写器
	reader := &arthasWSReader{conn: conn, ctx: ctx}
	writer := &arthasWSWriter{conn: conn, mu: &sync.Mutex{}}

	h.sendWSMessage(conn, "info", fmt.Sprintf("开始执行命令: %s", command))

	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  reader,
		Stdout: writer,
		Stderr: writer,
		Tty:    true,
	})

	if err != nil {
		if ctx.Err() != nil {
			h.sendWSMessage(conn, "info", "命令已停止")
		} else {
			h.sendWSError(conn, "执行命令失败: "+err.Error())
		}
	}
}

// sendWSMessage 发送WebSocket消息
func (h *ArthasHandler) sendWSMessage(conn *websocket.Conn, msgType string, content string) {
	msg := map[string]string{
		"type":    msgType,
		"content": content,
	}
	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)
}

// sendWSError 发送WebSocket错误消息
func (h *ArthasHandler) sendWSError(conn *websocket.Conn, errMsg string) {
	h.sendWSMessage(conn, "error", errMsg)
}

// arthasWSReader WebSocket读取器
type arthasWSReader struct {
	conn *websocket.Conn
	ctx  context.Context
}

func (r *arthasWSReader) Read(p []byte) (n int, err error) {
	select {
	case <-r.ctx.Done():
		return 0, io.EOF
	default:
		// 对于Arthas命令，通常不需要交互式输入
		// 如果需要停止，通过context取消
		<-r.ctx.Done()
		return 0, io.EOF
	}
}

// arthasWSWriter WebSocket写入器
type arthasWSWriter struct {
	conn *websocket.Conn
	mu   *sync.Mutex
}

func (w *arthasWSWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	msg := map[string]string{
		"type":    "output",
		"content": string(p),
	}
	data, _ := json.Marshal(msg)

	if err := w.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return 0, err
	}
	return len(p), nil
}

// GenerateFlameGraph 生成火焰图
func (h *ArthasHandler) GenerateFlameGraph(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Query("namespace")
	pod := c.Query("pod")
	container := c.Query("container")
	processID := c.Query("processId")
	duration := c.DefaultQuery("duration", "30")
	event := c.DefaultQuery("event", "cpu")      // cpu, alloc, lock, wall
	threadId := c.Query("threadId")              // 可选，指定线程ID
	includeThreads := c.Query("includeThreads")  // 是否按线程分组显示

	if clusterIDStr == "" || namespace == "" || pod == "" || container == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要参数",
		})
		return
	}

	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 构建 profiler 命令 - 输出到临时文件
	outputFile := "/tmp/arthas_flamegraph.html"

	// 构建 profiler 选项
	profilerOpts := fmt.Sprintf("-d %s -e %s -f %s", duration, event, outputFile)

	// 如果指定了线程ID
	if threadId != "" {
		profilerOpts = fmt.Sprintf("-d %s -e %s -t %s -f %s", duration, event, threadId, outputFile)
	}

	// 是否按线程分组
	if includeThreads == "true" {
		profilerOpts += " --threads"
	}

	// 使用专门的 profiler 命令构建脚本
	arthasCmd := h.buildArthasProfilerCommand(processID, profilerOpts, outputFile)

	// 执行命令（根据持续时间设置超时，profiler 需要额外的启动和输出时间）
	durationInt, _ := strconv.Atoi(duration)
	timeout := time.Duration(durationInt+60) * time.Second
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	output, err := h.execCommand(ctx, uint(clusterID), currentUserID.(uint), namespace, pod, container, arthasCmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "生成火焰图失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    output,
	})
}

// buildArthasProfilerCommand 构建 Arthas profiler 命令脚本
func (h *ArthasHandler) buildArthasProfilerCommand(processID string, profilerOpts string, outputFile string) []string {
	// 使用 arthas-boot.jar 执行 profiler 命令
	// profiler 命令会阻塞直到采样完成，然后输出到文件
	script := fmt.Sprintf(`
# 下载 arthas-boot.jar 如果不存在
if [ ! -f /tmp/arthas-boot.jar ]; then
    echo "[INFO] Downloading arthas-boot.jar..."
    curl -sL -o /tmp/arthas-boot.jar https://arthas.aliyun.com/arthas-boot.jar 2>/dev/null || \
    wget -q -O /tmp/arthas-boot.jar https://arthas.aliyun.com/arthas-boot.jar 2>/dev/null
    if [ ! -f /tmp/arthas-boot.jar ]; then
        echo "[ERROR] Failed to download arthas-boot.jar"
        exit 1
    fi
fi

TARGET_PID=%s
PROFILER_OPTS="%s"
OUTPUT_FILE="%s"

echo "[INFO] Starting Arthas profiler on process $TARGET_PID with options: $PROFILER_OPTS"

# 查找可用的 JDK
find_jdk() {
    JAVA_PATHS="/usr/lib/jvm /opt/java /opt/jdk /usr/java /opt /Library/Java/JavaVirtualMachines"
    for base in $JAVA_PATHS; do
        if [ -d "$base" ]; then
            for java_bin in $(find "$base" -name "java" -type f 2>/dev/null | grep -E "/bin/java$" | head -10); do
                java_home=$(dirname $(dirname "$java_bin"))
                if [ -f "$java_home/lib/tools.jar" ] || [ -d "$java_home/jmods" ]; then
                    echo "$java_bin"
                    return 0
                fi
            done
        fi
    done
    if [ -n "$JAVA_HOME" ]; then
        if [ -f "$JAVA_HOME/lib/tools.jar" ] || [ -d "$JAVA_HOME/jmods" ]; then
            echo "$JAVA_HOME/bin/java"
            return 0
        fi
    fi
    return 1
}

JAVA_BIN=$(find_jdk)
if [ -z "$JAVA_BIN" ]; then
    echo "[ERROR] 未找到 JDK 环境，Arthas 需要完整的 JDK"
    exit 1
fi
echo "[INFO] Using Java: $JAVA_BIN"

# 删除旧的输出文件
rm -f "$OUTPUT_FILE" 2>/dev/null

# 使用固定端口，基于进程ID计算
BASE_PORT=$((3658 + (TARGET_PID %% 100)))

# 等待端口就绪的函数
wait_for_port() {
    local port=$1
    local max_wait=20
    local waited=0
    while [ $waited -lt $max_wait ]; do
        if (echo > /dev/tcp/127.0.0.1/$port) 2>/dev/null; then
            echo "[INFO] Port $port is ready"
            return 0
        fi
        sleep 1
        waited=$((waited + 1))
        echo "[INFO] Waiting for telnet port $port... ($waited/$max_wait)"
    done
    return 1
}

# 执行 profiler 的函数
execute_profiler() {
    local port=$1
    local retry_count=0
    local max_retries=3

    while [ $retry_count -lt $max_retries ]; do
        retry_count=$((retry_count + 1))
        echo "[INFO] Executing profiler command (attempt $retry_count/$max_retries) on port $port..."

        PROFILER_CMD="profiler start $PROFILER_OPTS"
        OUTPUT=$($JAVA_BIN -jar /tmp/arthas-boot.jar $TARGET_PID --telnet-port $port --http-port -1 -c "$PROFILER_CMD" 2>&1)

        # 检查是否是端口冲突
        if echo "$OUTPUT" | grep -qE "telnet port.*is used|process detection timeout|unexpected process"; then
            echo "[WARN] Port $port conflict, trying different port..."
            port=$((port + 1))
            sleep 2
            continue
        fi

        # 检查是否 attach 成功但连接失败
        if echo "$OUTPUT" | grep -q "Attach process.*success" && echo "$OUTPUT" | grep -qE "Connection refused|Connect.*error"; then
            echo "[INFO] Agent attached, waiting for telnet server..."
            if wait_for_port $port; then
                # 重试执行命令
                OUTPUT=$($JAVA_BIN -jar /tmp/arthas-boot.jar $TARGET_PID --telnet-port $port --http-port -1 -c "$PROFILER_CMD" 2>&1)
            fi
        fi

        echo "[INFO] Profiler command output:"
        echo "$OUTPUT"

        # 检查输出文件是否生成
        if [ -f "$OUTPUT_FILE" ]; then
            echo "[INFO] Flame graph generated successfully"
            echo "---FLAMEGRAPH_START---"
            cat "$OUTPUT_FILE"
            echo "---FLAMEGRAPH_END---"
            return 0
        fi

        # 尝试使用 profiler stop 获取结果
        echo "[INFO] Output file not found, trying profiler stop..."
        STOP_OUTPUT=$($JAVA_BIN -jar /tmp/arthas-boot.jar $TARGET_PID --telnet-port $port --http-port -1 -c "profiler stop -f $OUTPUT_FILE" 2>&1)
        echo "$STOP_OUTPUT"

        if [ -f "$OUTPUT_FILE" ]; then
            echo "[INFO] Flame graph generated via stop command"
            echo "---FLAMEGRAPH_START---"
            cat "$OUTPUT_FILE"
            echo "---FLAMEGRAPH_END---"
            return 0
        fi

        sleep 3
    done

    return 1
}

# 执行 profiler
if execute_profiler $BASE_PORT; then
    exit 0
fi

echo "[ERROR] Failed to generate flame graph"
echo "[HINT] 可能的原因:"
echo "  1. 容器中没有完整的 JDK 环境"
echo "  2. Arthas telnet 服务启动失败"
echo "  3. 目标进程不支持 profiler"
exit 1
`, processID, profilerOpts, outputFile)

	return []string{"sh", "-c", script}
}

// CheckArthasInstalled 检查Arthas是否已安装
func (h *ArthasHandler) CheckArthasInstalled(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Query("namespace")
	pod := c.Query("pod")
	container := c.Query("container")

	if clusterIDStr == "" || namespace == "" || pod == "" || container == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要参数",
		})
		return
	}

	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 检查Java是否存在 - 尝试多种方式检测
	// 方式1: 直接运行 java -version
	// 方式2: 检查 JAVA_HOME
	// 方式3: 检查常见路径
	// 方式4: 检查是否有 Java 进程在运行
	javaCheckScript := `
# 方式1: 尝试 java -version
if java -version 2>&1 | head -3 | grep -qiE "java version|openjdk version|runtime environment"; then
    java -version 2>&1 | head -3
    exit 0
fi

# 方式2: 通过 JAVA_HOME
if [ -n "$JAVA_HOME" ] && [ -x "$JAVA_HOME/bin/java" ]; then
    $JAVA_HOME/bin/java -version 2>&1 | head -3
    exit 0
fi

# 方式3: 检查常见路径
for java_path in /usr/bin/java /usr/local/bin/java /opt/java/openjdk/bin/java /opt/jdk/bin/java; do
    if [ -x "$java_path" ]; then
        $java_path -version 2>&1 | head -3
        exit 0
    fi
done

# 方式4: 检查是否有 Java 进程（如果有 Java 进程，说明有 Java 环境）
if ps aux 2>/dev/null | grep -v grep | grep -q java; then
    echo "Java process detected"
    exit 0
fi

if jps 2>/dev/null | grep -v Jps | grep -q .; then
    echo "Java process detected via jps"
    exit 0
fi

# 方式5: 尝试 find 查找 java
java_bin=$(find /usr /opt -name "java" -type f 2>/dev/null | head -1)
if [ -n "$java_bin" ] && [ -x "$java_bin" ]; then
    $java_bin -version 2>&1 | head -3
    exit 0
fi

exit 1
`
	output, err := h.execCommand(c.Request.Context(), uint(clusterID), currentUserID.(uint), namespace, pod, container, []string{"sh", "-c", javaCheckScript})

	// 检查输出是否包含 Java 版本信息或 Java 进程检测结果
	hasJava := err == nil && (strings.Contains(strings.ToLower(output), "java version") ||
		strings.Contains(strings.ToLower(output), "openjdk version") ||
		strings.Contains(strings.ToLower(output), "runtime environment") ||
		strings.Contains(strings.ToLower(output), "java process detected"))

	javaVersion := ""
	if hasJava {
		javaVersion = strings.TrimSpace(output)
	}

	// 检查Arthas是否已下载
	arthasOutput, _ := h.execCommand(c.Request.Context(), uint(clusterID), currentUserID.(uint), namespace, pod, container, []string{"sh", "-c", "ls -la /tmp/arthas-boot.jar 2>/dev/null"})
	hasArthas := strings.Contains(arthasOutput, "arthas-boot.jar")

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"hasJava":     hasJava,
			"hasArthas":   hasArthas,
			"javaVersion": javaVersion,
		},
	})
}

// InstallArthas 安装Arthas
func (h *ArthasHandler) InstallArthas(c *gin.Context) {
	var req struct {
		ClusterID uint   `json:"clusterId" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
		Pod       string `json:"pod" binding:"required"`
		Container string `json:"container" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 下载Arthas
	script := `
cd /tmp && \
(curl -o arthas-boot.jar https://arthas.aliyun.com/arthas-boot.jar 2>/dev/null || \
wget -O arthas-boot.jar https://arthas.aliyun.com/arthas-boot.jar 2>/dev/null) && \
ls -la arthas-boot.jar
`

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	output, err := h.execCommand(ctx, req.ClusterID, currentUserID.(uint), req.Namespace, req.Pod, req.Container, []string{"sh", "-c", script})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "安装Arthas失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    output,
	})
}
