package asset

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// HostInfo 主机信息
type HostInfo struct {
	ID           uint
	IP           string
	Port         int
	SSHUser      string
	CredentialID uint
}

// CredentialInfo 凭证信息
type CredentialInfo struct {
	ID         uint
	Type       string
	Username   string
	Password   string
	PrivateKey string
}

// TerminalSession 终端会话
type TerminalSession struct {
	ID          string
	HostID      uint
	HostName    string
	HostIP      string
	UserID      uint
	Username    string
	SSHClient   *ssh.Client
	SSHSession  *ssh.Session
	StdinPipe   io.WriteCloser
	StdoutPipe  io.Reader
	StderrPipe  io.Reader
	Recorder    *AsciinemaRecorder // 录制器
	CreatedAt   time.Time
}

// TerminalManager 终端管理器
type TerminalManager struct {
	sessions    map[string]*TerminalSession
	mu          sync.RWMutex
	hostUseCase *assetbiz.HostUseCase
	db          *gorm.DB
}

// NewTerminalManager 创建终端管理器
func NewTerminalManager(hostUseCase *assetbiz.HostUseCase, db *gorm.DB) *TerminalManager {
	return &TerminalManager{
		sessions:    make(map[string]*TerminalSession),
		hostUseCase: hostUseCase,
		db:          db,
	}
}

// CreateSession 创建SSH会话
func (tm *TerminalManager) CreateSession(ctx context.Context, hostID uint, userID uint, username string, cols, rows uint16) (*TerminalSession, error) {
	// 获取主机信息
	hostVO, err := tm.hostUseCase.GetByID(ctx, hostID)
	if err != nil {
		return nil, fmt.Errorf("获取主机信息失败: %w", err)
	}

	// 获取凭证（需要解密后的凭证）
	var credential *assetbiz.Credential
	if hostVO.CredentialID > 0 {
		credentialRepo := tm.hostUseCase.GetCredentialRepo()
		credential, err = credentialRepo.GetByIDDecrypted(ctx, hostVO.CredentialID)
		if err != nil {
			return nil, fmt.Errorf("获取凭证信息失败: %w", err)
		}
	} else {
		return nil, fmt.Errorf("主机未配置凭证")
	}

	// 解析私钥
	var signer ssh.Signer
	var authMethod ssh.AuthMethod

	if credential.Type == "key" {
		// 使用私钥认证
		if credential.Password != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(credential.PrivateKey), []byte(credential.Password))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(credential.PrivateKey))
		}
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %w", err)
		}
		authMethod = ssh.PublicKeys(signer)
	} else {
		// 使用密码认证
		authMethod = ssh.Password(credential.Password)
	}

	// SSH配置
	config := &ssh.ClientConfig{
		User:            hostVO.SSHUser,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// 连接SSH
	address := fmt.Sprintf("%s:%d", hostVO.IP, hostVO.Port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %w", err)
	}

	// 创建会话
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("创建SSH会话失败: %w", err)
	}

	// 设置终端模式
	modes := ssh.TerminalModes{
		ssh.ECHO:   1, // 启用回显
		ssh.TTY_OP_ISPEED: 14400, // 输入速度
		ssh.TTY_OP_OSPEED: 14400, // 输出速度
	}

	if err := session.RequestPty("xterm-256color", int(rows), int(cols), modes); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("请求伪终端失败: %w", err)
	}

	// 获取管道
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("获取stdin管道失败: %w", err)
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("获取stdout管道失败: %w", err)
	}

	stderrPipe, err := session.StderrPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("获取stderr管道失败: %w", err)
	}

	// 启动shell
	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("启动shell失败: %w", err)
	}

	// 创建录制器
	recordingDir := "./data/terminal-recordings"
	recorder, err := NewAsciinemaRecorder(recordingDir, int(cols), int(rows))
	if err != nil {
		fmt.Printf("创建录制器失败: %v\n", err)
		// 录制失败不影响终端连接，继续
		recorder = nil
	}

	// 创建会话对象
	terminalSession := &TerminalSession{
		ID:         fmt.Sprintf("%d-%d", hostID, time.Now().Unix()),
		HostID:     hostID,
		HostName:   hostVO.Name,
		HostIP:     hostVO.IP,
		UserID:     userID,
		Username:   username,
		SSHClient:  client,
		SSHSession: session,
		StdinPipe:  stdinPipe,
		StdoutPipe: stdoutPipe,
		StderrPipe: stderrPipe,
		Recorder:   recorder,
		CreatedAt:  time.Now(),
	}

	// 保存会话
	tm.mu.Lock()
	tm.sessions[terminalSession.ID] = terminalSession
	tm.mu.Unlock()

	return terminalSession, nil
}

// GetSession 获取会话
func (tm *TerminalManager) GetSession(sessionID string) (*TerminalSession, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	session, ok := tm.sessions[sessionID]
	return session, ok
}

// CloseSession 关闭会话
func (tm *TerminalManager) CloseSession(sessionID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	session, ok := tm.sessions[sessionID]
	if !ok {
		appLogger.Warn("尝试关闭不存在的会话", zap.String("sessionID", sessionID))
		return fmt.Errorf("会话不存在")
	}

	appLogger.Info("开始关闭终端会话",
		zap.String("sessionID", sessionID),
		zap.Uint("hostID", session.HostID),
		zap.String("hostName", session.HostName),
		zap.Uint("userID", session.UserID),
		zap.String("username", session.Username))

	// 关闭录制器并保存会话信息
	if session.Recorder != nil {
		// 关闭录制器
		if err := session.Recorder.Close(); err != nil {
			appLogger.Error("关闭录制器失败", zap.Error(err))
		}

		// 获取录制信息
		duration := session.Recorder.GetDuration()
		fileSize := session.Recorder.GetFileSize()
		recordingPath := session.Recorder.GetRecordingPath()

		appLogger.Info("录制信息",
			zap.String("recordingPath", recordingPath),
			zap.Int("duration", duration),
			zap.Int64("fileSize", fileSize))

		// 保存会话记录到数据库
		terminalSession := &assetbiz.TerminalSession{
			HostID:        session.HostID,
			HostName:      session.HostName,
			HostIP:        session.HostIP,
			UserID:        session.UserID,
			Username:      session.Username,
			RecordingPath: recordingPath,
			Duration:      duration,
			FileSize:      fileSize,
			Status:        "completed",
		}

		appLogger.Info("准备保存终端会话记录到数据库",
			zap.Uint("hostID", terminalSession.HostID),
			zap.String("hostName", terminalSession.HostName),
			zap.Uint("userID", terminalSession.UserID),
			zap.String("username", terminalSession.Username))

		if err := tm.db.Create(terminalSession).Error; err != nil {
			appLogger.Error("保存终端会话记录失败",
				zap.Error(err),
				zap.Uint("hostID", session.HostID),
				zap.Uint("userID", session.UserID),
				zap.String("recordingPath", recordingPath))
		} else {
			appLogger.Info("终端会话记录已成功保存到数据库",
				zap.Uint("sessionID", terminalSession.ID),
				zap.String("username", terminalSession.Username),
				zap.String("hostName", terminalSession.HostName),
				zap.Int("duration", duration))
		}
	} else {
		appLogger.Warn("会话没有录制器", zap.String("sessionID", sessionID))
	}

	// 关闭SSH连接
	if session.SSHSession != nil {
		session.SSHSession.Close()
	}
	if session.SSHClient != nil {
		session.SSHClient.Close()
	}

	delete(tm.sessions, sessionID)
	appLogger.Info("终端会话已关闭", zap.String("sessionID", sessionID))
	return nil
}

// HandleSSHConnection 处理SSH WebSocket连接
func (s *HTTPServer) HandleSSHConnection(c *gin.Context) {
	hostIdStr := c.Param("id")
	hostId, err := strconv.Atoi(hostIdStr)
	if err != nil {
		appLogger.Error("无效的主机ID", zap.String("hostId", hostIdStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的主机ID"})
		return
	}

	// 获取用户信息（从context中获取，假设已经在中间件中设置）
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	var uid uint = 0
	var uname string = "unknown"

	if userID != nil {
		if id, ok := userID.(uint); ok {
			uid = id
		} else if id, ok := userID.(float64); ok {
			uid = uint(id)
		}
	}
	if username != nil {
		if name, ok := username.(string); ok {
			uname = name
		}
	}

	appLogger.Info("WebSocket终端连接请求",
		zap.Int("hostId", hostId),
		zap.Uint("userId", uid),
		zap.String("username", uname))

	// 从URL参数读取终端尺寸，默认80x24
	colsStr := c.DefaultQuery("cols", "80")
	rowsStr := c.DefaultQuery("rows", "24")
	cols, err := strconv.ParseUint(colsStr, 10, 16)
	if err != nil || cols == 0 {
		cols = 80
	}
	rows, err := strconv.ParseUint(rowsStr, 10, 16)
	if err != nil || rows == 0 {
		rows = 24
	}

	appLogger.Info("终端尺寸参数",
		zap.Uint64("cols", cols),
		zap.Uint64("rows", rows),
		zap.String("colsStr", colsStr),
		zap.String("rowsStr", rowsStr))

	// 升级到WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		appLogger.Error("WebSocket升级失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WebSocket升级失败"})
		return
	}
	defer conn.Close()

	// 设置读取超时，确保连接断开时能及时检测
	conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		return nil
	})

	// 创建SSH会话
	session, err := s.terminalManager.CreateSession(c.Request.Context(), uint(hostId), uid, uname, uint16(cols), uint16(rows))
	if err != nil {
		appLogger.Error("SSH会话创建失败", zap.Error(err), zap.Int("hostId", hostId))
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("连接失败: %s\r\n", err.Error())))
		return
	}

	// 确保会话被关闭 - 使用显式调用而不是 defer
	sessionClosed := false
	closeSession := func() {
		if !sessionClosed {
			sessionClosed = true
			s.terminalManager.CloseSession(session.ID)
		}
	}
	defer closeSession()

	appLogger.Info("SSH会话创建成功", zap.String("sessionID", session.ID), zap.Int("hostId", hostId))

	// 启动goroutine从SSH读取输出并发送到WebSocket
	var wg sync.WaitGroup
	wg.Add(2)

	// 读取stdout
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := session.StdoutPipe.Read(buf)
			if n > 0 {
				// 录制输出
				if session.Recorder != nil {
					session.Recorder.RecordOutput(buf[:n])
				}
				// 使用二进制消息以保留原始字节（包括CR/LF控制字符）
				conn.WriteMessage(websocket.BinaryMessage, buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	// 读取stderr
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := session.StderrPipe.Read(buf)
			if n > 0 {
				// 录制输出
				if session.Recorder != nil {
					session.Recorder.RecordOutput(buf[:n])
				}
				// 使用二进制消息以保留原始字节（包括CR/LF控制字符）
				conn.WriteMessage(websocket.BinaryMessage, buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	// 处理来自WebSocket的消息并发送到SSH
	for {
		// 每次读取前更新超时时间
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		messageType, data, err := conn.ReadMessage()
		if err != nil {
			appLogger.Info("WebSocket连接关闭", zap.String("sessionID", session.ID), zap.Error(err))
			// 立即关闭SSH连接，让所有阻塞的Read操作返回
			if session.SSHSession != nil {
				session.SSHSession.Close()
			}
			if session.SSHClient != nil {
				session.SSHClient.Close()
			}
			break
		}

		if messageType == websocket.TextMessage {
			// 尝试解析JSON消息（可能是resize命令）
			var msg map[string]interface{}
			if err := json.Unmarshal(data, &msg); err == nil {
				if msgType, ok := msg["type"].(string); ok && msgType == "resize" {
					cols, colsOk := msg["cols"].(float64)
					rows, rowsOk := msg["rows"].(float64)
					if colsOk && rowsOk {
						// 调整SSH会话窗口大小
						if err := session.SSHSession.WindowChange(int(rows), int(cols)); err != nil {
							appLogger.Error("调整窗口大小失败", zap.Error(err))
						}
						continue
					}
				}
			}
			// 如果不是resize命令，当作普通输入发送到SSH
			// 录制输入
			if session.Recorder != nil {
				session.Recorder.RecordInput(data)
			}
			session.StdinPipe.Write(data)
		} else if messageType == websocket.BinaryMessage {
			// 录制输入
			if session.Recorder != nil {
				session.Recorder.RecordInput(data)
			}
			session.StdinPipe.Write(data)
		}
	}

	wg.Wait()
	appLogger.Info("终端会话结束", zap.String("sessionID", session.ID))
}

// ResizeTerminal 调整终端大小
func (s *HTTPServer) ResizeTerminal(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "终端大小调整功能待实现"})
}
