package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/internal/logagent"
	"github.com/ydcloud-dy/opshub/pkg/response"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type agentControlHandler struct {
	db *gorm.DB
}

type agentConfigStatusPayload struct {
	AgentID          string                  `json:"agentId"`
	AgentToken       string                  `json:"agentToken"`
	ConfigVersion    uint64                  `json:"configVersion"`
	ReloadGeneration uint64                  `json:"reloadGeneration"`
	Status           string                  `json:"status"`
	Error            string                  `json:"error"`
	Assignments      []agentAssignmentStatus `json:"assignments"`
}

type agentAssignmentStatus struct {
	PolicyID      uint   `json:"policyId"`
	PolicyVersion uint64 `json:"policyVersion"`
	Status        string `json:"status"`
	Error         string `json:"error"`
}

type agentLogHeartbeatPayload struct {
	AgentID       string                   `json:"agentId"`
	AgentToken    string                   `json:"agentToken"`
	Version       string                   `json:"version"`
	Hostname      string                   `json:"hostname"`
	Mode          string                   `json:"mode"`
	ConfigVersion uint64                   `json:"configVersion"`
	Metrics       logagent.MetricsSnapshot `json:"metrics"`
}

type agentLogConfigResponse struct {
	ConfigVersion    uint64                   `json:"configVersion"`
	ReloadGeneration uint64                   `json:"reloadGeneration"`
	PollInterval     int                      `json:"pollInterval"`
	LogCollection    logagent.Config          `json:"logCollection"`
	Assignments      []agentDesiredAssignment `json:"assignments"`
}

type agentDesiredAssignment struct {
	PolicyID      uint   `json:"policyId"`
	PolicyVersion uint64 `json:"policyVersion"`
	DesiredState  string `json:"desiredState"`
}

const agentConfigETagSchema = "20260722-1"

type agentConfigETagPayload struct {
	Schema            string                   `json:"schema"`
	ConfigVersion     uint64                   `json:"configVersion"`
	ReloadGeneration  uint64                   `json:"reloadGeneration"`
	PollInterval      int                      `json:"pollInterval"`
	Enabled           bool                     `json:"enabled"`
	GatewayURL        string                   `json:"gatewayUrl"`
	GatewayTokenHash  string                   `json:"gatewayTokenHash"`
	DesiredAssignment []agentDesiredAssignment `json:"assignments"`
}

func buildAgentConfigETag(result agentLogConfigResponse) string {
	tokenHash := sha256.Sum256([]byte(result.LogCollection.GatewayToken))
	payload := agentConfigETagPayload{
		Schema: agentConfigETagSchema, ConfigVersion: result.ConfigVersion,
		ReloadGeneration: result.ReloadGeneration, PollInterval: result.PollInterval,
		Enabled: result.LogCollection.Enabled, GatewayURL: result.LogCollection.GatewayURL,
		GatewayTokenHash: hex.EncodeToString(tokenHash[:]), DesiredAssignment: result.Assignments,
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func RegisterPublicRoutes(router *gin.RouterGroup, db *gorm.DB) {
	handler := &agentControlHandler{db: db}
	agents := router.Group("/agents")
	{
		agents.GET("/log-config", handler.GetLogConfig)
		agents.POST("/log-config/status", handler.ReportLogConfigStatus)
		agents.POST("/log-heartbeat", handler.ReportLogHeartbeat)
	}
	handler.registerKubernetesCollectorRoutes(router)
}

func (h *agentControlHandler) GetLogConfig(c *gin.Context) {
	host, ok := h.authenticate(c, c.Query("agentId"), "")
	if !ok {
		return
	}

	var result agentLogConfigResponse
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		instance, err := syncAgentAssignments(tx, host)
		if err != nil {
			return err
		}
		result, err = buildAgentLogConfig(tx, c, host, instance)
		return err
	})
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "生成日志采集配置失败: "+err.Error())
		return
	}

	etag := buildAgentConfigETag(result)
	c.Header("ETag", etag)
	c.Header("Cache-Control", "no-cache")
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	response.Success(c, result)
}

func (h *agentControlHandler) ReportLogConfigStatus(c *gin.Context) {
	var payload agentConfigStatusPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	host, ok := h.authenticate(c, payload.AgentID, payload.AgentToken)
	if !ok {
		return
	}
	now := time.Now()
	status := normalizeApplyStatus(payload.Status)
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"host_id": host.ID, "hostname": firstNonEmpty(host.Hostname, host.Name),
			"status": "online", "config_version": payload.ConfigVersion,
			"last_heartbeat_at": &now, "last_error": strings.TrimSpace(payload.Error),
		}
		if err := tx.Model(&logmodel.CollectorInstance{}).Where("instance_id = ?", host.AgentID).Updates(updates).Error; err != nil {
			return err
		}
		for _, assignment := range payload.Assignments {
			assignmentStatus := normalizeApplyStatus(assignment.Status)
			values := map[string]any{
				"apply_status": assignmentStatus, "last_error": strings.TrimSpace(assignment.Error),
			}
			if assignment.PolicyVersion > 0 {
				values["policy_version"] = assignment.PolicyVersion
			}
			if assignmentStatus == "applied" || assignmentStatus == "disabled" {
				values["applied_at"] = &now
			} else {
				values["applied_at"] = nil
			}
			if err := tx.Model(&logmodel.CollectorAssignment{}).
				Where("instance_id = ? AND policy_id = ?", host.AgentID, assignment.PolicyID).
				Updates(values).Error; err != nil {
				return err
			}
		}
		if len(payload.Assignments) == 0 && status != "" {
			values := map[string]any{"apply_status": status, "last_error": strings.TrimSpace(payload.Error)}
			if status == "applied" || status == "disabled" {
				values["applied_at"] = &now
			}
			return tx.Model(&logmodel.CollectorAssignment{}).Where("instance_id = ? AND desired_state = ?", host.AgentID, "active").Updates(values).Error
		}
		return nil
	})
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存配置应用状态失败: "+err.Error())
		return
	}
	response.SuccessWithMessage(c, "log config status accepted", nil)
}

func (h *agentControlHandler) ReportLogHeartbeat(c *gin.Context) {
	var payload agentLogHeartbeatPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	host, ok := h.authenticate(c, payload.AgentID, payload.AgentToken)
	if !ok {
		return
	}
	now := time.Now()
	var existing logmodel.CollectorInstance
	if err := h.db.WithContext(c.Request.Context()).Where("instance_id = ?", host.AgentID).First(&existing).Error; err != nil && err != gorm.ErrRecordNotFound {
		response.ErrorCode(c, http.StatusInternalServerError, "读取采集实例失败: "+err.Error())
		return
	}
	inputEPS, outputEPS := calculateAgentEPS(existing, payload.Metrics, now)
	metricsRaw, _ := json.Marshal(payload.Metrics)
	lastIngestAt := payload.Metrics.LastSuccess
	instance := logmodel.CollectorInstance{
		InstanceID: host.AgentID, AgentID: host.AgentID, Mode: firstNonEmpty(payload.Mode, "host"),
		HostID: host.ID, Hostname: firstNonEmpty(payload.Hostname, host.Hostname, host.Name),
		CollectorType: "opshub-agent", Version: firstNonEmpty(payload.Version, host.AgentVersion),
		ConfigVersion: payload.ConfigVersion, Status: "online", LastHeartbeatAt: &now,
		WALBytes: payload.Metrics.WALBytes, InputEPS: inputEPS, OutputEPS: outputEPS,
		DroppedTotal: payload.Metrics.DroppedRecords, RetryTotal: payload.Metrics.RetryTotal,
		LastError: payload.Metrics.LastError, Metrics: string(metricsRaw),
	}
	if !lastIngestAt.IsZero() {
		instance.LastIngestAt = &lastIngestAt
	}
	updates := map[string]any{
		"agent_id": instance.AgentID, "mode": instance.Mode, "host_id": instance.HostID,
		"hostname": instance.Hostname, "collector_type": instance.CollectorType, "version": instance.Version,
		"config_version": instance.ConfigVersion, "status": instance.Status, "last_heartbeat_at": instance.LastHeartbeatAt,
		"wal_bytes": instance.WALBytes, "input_eps": instance.InputEPS,
		"output_eps": instance.OutputEPS, "dropped_total": instance.DroppedTotal, "retry_total": instance.RetryTotal,
		"last_error": instance.LastError, "metrics": instance.Metrics,
	}
	if instance.LastIngestAt != nil {
		updates["last_ingest_at"] = instance.LastIngestAt
	}
	if err := h.db.WithContext(c.Request.Context()).Where("instance_id = ?", host.AgentID).Assign(updates).FirstOrCreate(&instance).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存采集心跳失败: "+err.Error())
		return
	}
	response.SuccessWithMessage(c, "log heartbeat accepted", nil)
}

func (h *agentControlHandler) authenticate(c *gin.Context, agentID, token string) (assetbiz.Host, bool) {
	agentID = firstNonEmpty(strings.TrimSpace(agentID), strings.TrimSpace(c.GetHeader("X-OpsHub-Agent-ID")))
	if token == "" {
		token = bearerToken(c.GetHeader("Authorization"))
	}
	if token == "" {
		token = strings.TrimSpace(c.GetHeader("X-OpsHub-Agent-Token"))
	}
	if agentID == "" || token == "" {
		response.ErrorCode(c, http.StatusUnauthorized, "Agent 认证信息不能为空")
		return assetbiz.Host{}, false
	}
	var host assetbiz.Host
	if err := h.db.WithContext(c.Request.Context()).Where("agent_id = ?", agentID).First(&host).Error; err != nil {
		response.ErrorCode(c, http.StatusUnauthorized, "Agent 不存在")
		return host, false
	}
	actual := sha256.Sum256([]byte(token))
	actualHex := hex.EncodeToString(actual[:])
	if subtle.ConstantTimeCompare([]byte(host.AgentTokenHash), []byte(actualHex)) != 1 {
		response.ErrorCode(c, http.StatusUnauthorized, "Agent 认证失败")
		return host, false
	}
	return host, true
}

func syncAgentAssignments(tx *gorm.DB, host assetbiz.Host) (logmodel.CollectorInstance, error) {
	now := time.Now()
	instance := logmodel.CollectorInstance{InstanceID: host.AgentID}
	if err := tx.Where("instance_id = ?", host.AgentID).Assign(map[string]any{
		"agent_id": host.AgentID, "mode": "host", "host_id": host.ID,
		"hostname": firstNonEmpty(host.Hostname, host.Name), "collector_type": "opshub-agent",
		"version": host.AgentVersion, "status": "online", "last_heartbeat_at": &now,
	}).FirstOrCreate(&instance).Error; err != nil {
		return instance, err
	}

	var policyIDs []uint
	query := tx.Model(&logmodel.PolicyTarget{}).Distinct("policy_id").
		Where("(target_type = ? AND target_id = ?) OR (target_type = ? AND target_id = ?)", "host", host.ID, "host_group", host.GroupID)
	if err := query.Pluck("policy_id", &policyIDs).Error; err != nil {
		return instance, err
	}
	var policies []logmodel.CollectionPolicy
	if len(policyIDs) > 0 {
		if err := tx.Where("id IN ? AND status = ?", policyIDs, "published").Find(&policies).Error; err != nil {
			return instance, err
		}
	}
	activeIDs := make([]uint, 0, len(policies))
	for _, policy := range policies {
		activeIDs = append(activeIDs, policy.ID)
		assignment := logmodel.CollectorAssignment{
			InstanceID: host.AgentID, PolicyID: policy.ID, PolicyVersion: policy.Version,
			DesiredState: "active", ApplyStatus: "pending",
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "instance_id"}, {Name: "policy_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"policy_version": policy.Version, "desired_state": "active",
			}),
		}).Create(&assignment).Error; err != nil {
			return instance, err
		}
	}
	disableQuery := tx.Model(&logmodel.CollectorAssignment{}).Where("instance_id = ?", host.AgentID)
	if len(activeIDs) > 0 {
		disableQuery = disableQuery.Where("policy_id NOT IN ?", activeIDs)
	}
	disableQuery = disableQuery.Where("desired_state <> ?", "disabled")
	if err := disableQuery.Updates(map[string]any{"desired_state": "disabled", "apply_status": "pending", "applied_at": nil}).Error; err != nil {
		return instance, err
	}
	return instance, nil
}

func buildAgentLogConfig(tx *gorm.DB, c *gin.Context, host assetbiz.Host, instance logmodel.CollectorInstance) (agentLogConfigResponse, error) {
	var assignments []logmodel.CollectorAssignment
	if err := tx.Where("instance_id = ?", host.AgentID).Order("policy_id ASC").Find(&assignments).Error; err != nil {
		return agentLogConfigResponse{}, err
	}
	config := logagent.Config{
		Enabled: false, GatewayURL: resolveAgentGatewayURL(c), GatewayToken: resolveAgentGatewayToken(c),
		StateDir: "/var/lib/opshub-agent/logs", ScanIntervalSeconds: 1, BatchSize: 500,
		FlushIntervalSeconds: 2, MaxWALBytes: 2 * 1024 * 1024 * 1024,
	}
	desiredAssignments := make([]agentDesiredAssignment, 0, len(assignments))
	var configVersion uint64
	for _, assignment := range assignments {
		desiredAssignments = append(desiredAssignments, agentDesiredAssignment{
			PolicyID: assignment.PolicyID, PolicyVersion: assignment.PolicyVersion, DesiredState: assignment.DesiredState,
		})
		if assignment.DesiredState != "active" {
			continue
		}
		var revision logmodel.PolicyRevision
		if err := tx.Where("policy_id = ? AND version = ?", assignment.PolicyID, assignment.PolicyVersion).First(&revision).Error; err != nil {
			return agentLogConfigResponse{}, fmt.Errorf("策略 %d v%d 缺少发布版本", assignment.PolicyID, assignment.PolicyVersion)
		}
		var payload policyPayload
		if err := json.Unmarshal([]byte(revision.Content), &payload); err != nil {
			return agentLogConfigResponse{}, fmt.Errorf("策略 %d v%d 内容无效: %w", assignment.PolicyID, assignment.PolicyVersion, err)
		}
		payload.normalize()
		config.Sources = append(config.Sources, logagent.SourceConfig{
			ID: fmt.Sprintf("policy-%d", assignment.PolicyID), PolicyID: uint64(assignment.PolicyID), PolicyVersion: assignment.PolicyVersion,
			Paths: payload.Paths, ExcludePaths: payload.ExcludePaths, ReadFrom: payload.ReadFrom,
			Encoding: payload.Encoding, Environment: payload.Environment, Service: payload.Service,
			Stream: payload.Stream, MaxLineBytes: payload.MaxLineBytes, Parser: payload.Parser, Multiline: payload.Multiline,
			Redaction: payload.Redaction, Retention: payload.Retention,
		})
		if payload.WALMaxBytes > config.MaxWALBytes {
			config.MaxWALBytes = payload.WALMaxBytes
		}
		if uint64(revision.ID) > configVersion {
			configVersion = uint64(revision.ID)
		}
	}
	config.Enabled = len(config.Sources) > 0
	config.Normalize()
	if config.Enabled {
		if config.GatewayURL == "" {
			return agentLogConfigResponse{}, fmt.Errorf("未配置 Agent 可访问的 Log Gateway 地址")
		}
		if config.GatewayToken == "" {
			return agentLogConfigResponse{}, fmt.Errorf("未配置 Log Gateway Agent Token")
		}
	}
	return agentLogConfigResponse{
		ConfigVersion: configVersion, ReloadGeneration: instance.ReloadGeneration,
		PollInterval: 30, LogCollection: config, Assignments: desiredAssignments,
	}, nil
}

func resolveAgentGatewayURL(c *gin.Context) string {
	publicScheme := firstNonEmpty(c.GetHeader("X-Forwarded-Proto"), os.Getenv("OPSHUB_PUBLIC_SCHEME"), "http")
	if configured := strings.TrimRight(strings.TrimSpace(os.Getenv("OPSHUB_LOG_GATEWAY_PUBLIC_URL")), "/"); configured != "" {
		return normalizePublicURL(configured, publicScheme)
	}
	for _, key := range []string{"OPSHUB_SERVER_FRONTEND_URL", "OPSHUB_SERVER_EXTERNAL_URL"} {
		if configured := strings.TrimRight(strings.TrimSpace(os.Getenv(key)), "/"); configured != "" {
			return normalizePublicURL(configured, publicScheme)
		}
	}
	if publicHost := strings.TrimSpace(os.Getenv("OPSHUB_PUBLIC_HOST")); publicHost != "" {
		return normalizePublicURL(publicHost, publicScheme)
	}
	host := firstNonEmpty(firstForwardedValue(c.GetHeader("X-Forwarded-Host")), c.Request.Host)
	return agentGatewayURLForHost(publicScheme, host, autoDetectedAgentGatewayURL())
}

func agentGatewayURLForHost(scheme, host, loopbackFallback string) string {
	parsed, err := url.Parse(firstNonEmpty(strings.TrimSpace(scheme), "http") + "://" + strings.TrimSpace(host))
	if err != nil || parsed.Hostname() == "" {
		return strings.TrimRight(loopbackFallback, "/")
	}
	hostname := parsed.Hostname()
	backendPort := firstNonEmpty(os.Getenv("OPSHUB_SERVER_HTTP_PORT"), "9876")
	fallbackHostname := ""
	if fallback, fallbackErr := url.Parse(strings.TrimSpace(loopbackFallback)); fallbackErr == nil {
		fallbackHostname = fallback.Hostname()
	}
	if strings.EqualFold(hostname, "localhost") ||
		(net.ParseIP(hostname) != nil && net.ParseIP(hostname).IsLoopback()) ||
		parsed.Port() == backendPort ||
		(fallbackHostname != "" && strings.EqualFold(hostname, fallbackHostname)) {
		return strings.TrimRight(loopbackFallback, "/")
	}
	return strings.TrimRight(parsed.Scheme+"://"+parsed.Host, "/")
}

func autoDetectedAgentGatewayURL() string {
	ip := firstNonLoopbackAgentIPv4()
	if ip == "" {
		return ""
	}
	port := firstNonEmpty(os.Getenv("OPSHUB_LOG_GATEWAY_PUBLIC_PORT"), "9880")
	return "http://" + net.JoinHostPort(ip, port)
}

func firstNonLoopbackAgentIPv4() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, networkInterface := range interfaces {
		if networkInterface.Flags&net.FlagUp == 0 || networkInterface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, addressErr := networkInterface.Addrs()
		if addressErr != nil {
			continue
		}
		for _, address := range addresses {
			ip, _, parseErr := net.ParseCIDR(address.String())
			if parseErr != nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

func firstForwardedValue(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.IndexByte(value, ','); index >= 0 {
		value = strings.TrimSpace(value[:index])
	}
	return value
}

func normalizePublicURL(value, scheme string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" || strings.Contains(value, "://") {
		return value
	}
	return firstNonEmpty(strings.TrimSpace(scheme), "http") + "://" + value
}

func resolveAgentGatewayToken(c *gin.Context) string {
	for _, key := range []string{"OPSHUB_LOG_AGENT_INGEST_TOKEN", "OPSHUB_LOG_INGEST_TOKEN", "OPSHUB_LOG_INGEST_TEST_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(key)); token != "" {
			return token
		}
	}
	gatewayURL := resolveAgentGatewayURL(c)
	parsed, err := url.Parse(gatewayURL)
	localGatewayURL := autoDetectedAgentGatewayURL()
	if err == nil && ((strings.EqualFold(parsed.Hostname(), "localhost") || (net.ParseIP(parsed.Hostname()) != nil && net.ParseIP(parsed.Hostname()).IsLoopback())) || strings.EqualFold(strings.TrimRight(gatewayURL, "/"), strings.TrimRight(localGatewayURL, "/"))) {
		return "opshub-local-ingest-token"
	}
	return ""
}

func calculateAgentEPS(existing logmodel.CollectorInstance, current logagent.MetricsSnapshot, now time.Time) (float64, float64) {
	if existing.LastHeartbeatAt == nil || existing.Metrics == "" {
		return 0, 0
	}
	var previous logagent.MetricsSnapshot
	if json.Unmarshal([]byte(existing.Metrics), &previous) != nil {
		return 0, 0
	}
	seconds := now.Sub(*existing.LastHeartbeatAt).Seconds()
	if seconds <= 0 {
		return 0, 0
	}
	inputDelta := float64(current.InputRecords - minUint64(current.InputRecords, previous.InputRecords))
	outputDelta := float64(current.OutputRecords - minUint64(current.OutputRecords, previous.OutputRecords))
	return math.Round(inputDelta/seconds*100) / 100, math.Round(outputDelta/seconds*100) / 100
}

func normalizeApplyStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "applied", "running", "success":
		return "applied"
	case "failed", "error":
		return "failed"
	case "pending", "applying":
		return "pending"
	case "disabled":
		return "disabled"
	default:
		return ""
	}
}

func bearerToken(value string) string {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func minUint64(left, right uint64) uint64 {
	if left < right {
		return left
	}
	return right
}
