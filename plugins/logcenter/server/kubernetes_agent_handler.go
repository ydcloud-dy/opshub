package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/logagent"
	"github.com/ydcloud-dy/opshub/pkg/response"
	k8smodel "github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type kubernetesCollectorIdentity struct {
	ClusterID  uint
	Cluster    k8smodel.Cluster
	NodeName   string
	InstanceID string
}

type kubernetesCollectorPayload struct {
	ClusterID        uint                     `json:"clusterId"`
	ClusterToken     string                   `json:"clusterToken"`
	NodeName         string                   `json:"nodeName"`
	PodName          string                   `json:"podName"`
	Namespace        string                   `json:"namespace"`
	Version          string                   `json:"version"`
	ConfigVersion    uint64                   `json:"configVersion"`
	ReloadGeneration uint64                   `json:"reloadGeneration"`
	Status           string                   `json:"status"`
	Error            string                   `json:"error"`
	Assignments      []agentAssignmentStatus  `json:"assignments"`
	Metrics          logagent.MetricsSnapshot `json:"metrics"`
}

func (h *agentControlHandler) registerKubernetesCollectorRoutes(router *gin.RouterGroup) {
	collectors := router.Group("/log-collectors/kubernetes")
	{
		collectors.POST("/register", h.RegisterKubernetesCollector)
		collectors.GET("/config", h.GetKubernetesLogConfig)
		collectors.POST("/config/status", h.ReportKubernetesLogConfigStatus)
		collectors.POST("/heartbeat", h.ReportKubernetesLogHeartbeat)
	}
}

func (h *agentControlHandler) RegisterKubernetesCollector(c *gin.Context) {
	var payload kubernetesCollectorPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	identity, ok := h.authenticateKubernetesCollector(c, payload.ClusterID, payload.NodeName, payload.ClusterToken)
	if !ok {
		return
	}
	instance, err := h.upsertKubernetesCollector(c, identity, payload)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "注册 Kubernetes 日志采集器失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{
		"instanceId": identity.InstanceID, "clusterId": identity.ClusterID,
		"clusterName": identity.Cluster.Name, "pollInterval": 30,
		"reloadGeneration": instance.ReloadGeneration,
	})
}

func (h *agentControlHandler) GetKubernetesLogConfig(c *gin.Context) {
	clusterID := uint(parseUint(c.Query("clusterId")))
	nodeName := strings.TrimSpace(c.Query("nodeName"))
	identity, ok := h.authenticateKubernetesCollector(c, clusterID, nodeName, "")
	if !ok {
		return
	}
	var result agentLogConfigResponse
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		instance, err := syncKubernetesAssignments(tx, identity)
		if err != nil {
			return err
		}
		result, err = buildKubernetesLogConfig(tx, c, identity, instance)
		return err
	})
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "生成 Kubernetes 日志采集配置失败: "+err.Error())
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

func (h *agentControlHandler) ReportKubernetesLogConfigStatus(c *gin.Context) {
	var payload kubernetesCollectorPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	identity, ok := h.authenticateKubernetesCollector(c, payload.ClusterID, payload.NodeName, payload.ClusterToken)
	if !ok {
		return
	}
	now := time.Now()
	status := normalizeApplyStatus(payload.Status)
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&logmodel.CollectorInstance{}).Where("instance_id = ?", identity.InstanceID).Updates(map[string]any{
			"status": "online", "config_version": payload.ConfigVersion,
			"last_heartbeat_at": &now, "last_error": strings.TrimSpace(payload.Error),
		}).Error; err != nil {
			return err
		}
		for _, assignment := range payload.Assignments {
			assignmentStatus := normalizeApplyStatus(assignment.Status)
			values := map[string]any{"apply_status": assignmentStatus, "last_error": strings.TrimSpace(assignment.Error)}
			if assignment.PolicyVersion > 0 {
				values["policy_version"] = assignment.PolicyVersion
			}
			if assignmentStatus == "applied" || assignmentStatus == "disabled" {
				values["applied_at"] = &now
			} else {
				values["applied_at"] = nil
			}
			if err := tx.Model(&logmodel.CollectorAssignment{}).Where("instance_id = ? AND policy_id = ?", identity.InstanceID, assignment.PolicyID).Updates(values).Error; err != nil {
				return err
			}
		}
		if len(payload.Assignments) == 0 && status != "" {
			return tx.Model(&logmodel.CollectorAssignment{}).Where("instance_id = ? AND desired_state = ?", identity.InstanceID, "active").Updates(map[string]any{
				"apply_status": status, "last_error": strings.TrimSpace(payload.Error),
			}).Error
		}
		return nil
	})
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存 Kubernetes 配置状态失败: "+err.Error())
		return
	}
	response.SuccessWithMessage(c, "kubernetes log config status accepted", nil)
}

func (h *agentControlHandler) ReportKubernetesLogHeartbeat(c *gin.Context) {
	var payload kubernetesCollectorPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	identity, ok := h.authenticateKubernetesCollector(c, payload.ClusterID, payload.NodeName, payload.ClusterToken)
	if !ok {
		return
	}
	instance, err := h.upsertKubernetesCollector(c, identity, payload)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存 Kubernetes 采集心跳失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"instanceId": instance.InstanceID})
}

func (h *agentControlHandler) authenticateKubernetesCollector(c *gin.Context, clusterID uint, nodeName, token string) (kubernetesCollectorIdentity, bool) {
	if clusterID == 0 {
		clusterID = uint(parseUint(c.GetHeader("X-OpsHub-Cluster-ID")))
	}
	nodeName = firstNonEmpty(strings.TrimSpace(nodeName), strings.TrimSpace(c.GetHeader("X-OpsHub-Node-Name")))
	if token == "" {
		token = bearerToken(c.GetHeader("Authorization"))
	}
	if clusterID == 0 || nodeName == "" || token == "" {
		response.ErrorCode(c, http.StatusUnauthorized, "集群、节点和采集 Token 不能为空")
		return kubernetesCollectorIdentity{}, false
	}
	var credential logmodel.ClusterCollectorCredential
	if err := h.db.WithContext(c.Request.Context()).Where("cluster_id = ? AND status = ?", clusterID, "active").First(&credential).Error; err != nil {
		response.ErrorCode(c, http.StatusUnauthorized, "Kubernetes 日志采集凭据不存在")
		return kubernetesCollectorIdentity{}, false
	}
	actual := sha256.Sum256([]byte(token))
	if subtle.ConstantTimeCompare([]byte(credential.TokenHash), []byte(hex.EncodeToString(actual[:]))) != 1 {
		response.ErrorCode(c, http.StatusUnauthorized, "Kubernetes 日志采集认证失败")
		return kubernetesCollectorIdentity{}, false
	}
	var cluster k8smodel.Cluster
	if err := h.db.WithContext(c.Request.Context()).First(&cluster, clusterID).Error; err != nil {
		response.ErrorCode(c, http.StatusUnauthorized, "Kubernetes 集群不存在")
		return kubernetesCollectorIdentity{}, false
	}
	return kubernetesCollectorIdentity{
		ClusterID: clusterID, Cluster: cluster, NodeName: nodeName,
		InstanceID: kubernetesCollectorInstanceID(clusterID, nodeName),
	}, true
}

func (h *agentControlHandler) upsertKubernetesCollector(c *gin.Context, identity kubernetesCollectorIdentity, payload kubernetesCollectorPayload) (logmodel.CollectorInstance, error) {
	now := time.Now()
	var existing logmodel.CollectorInstance
	if err := h.db.WithContext(c.Request.Context()).Where("instance_id = ?", identity.InstanceID).First(&existing).Error; err != nil && err != gorm.ErrRecordNotFound {
		return existing, err
	}
	inputEPS, outputEPS := calculateAgentEPS(existing, payload.Metrics, now)
	metricsRaw, _ := json.Marshal(payload.Metrics)
	updates := map[string]any{
		"agent_id": identity.InstanceID, "mode": "kubernetes-node", "host_id": 0,
		"cluster_id": identity.ClusterID, "hostname": identity.NodeName, "node_name": identity.NodeName,
		"pod_name": strings.TrimSpace(payload.PodName), "namespace": strings.TrimSpace(payload.Namespace),
		"collector_type": "opshub-agent", "version": payload.Version,
		"config_version": payload.ConfigVersion, "status": "online", "last_heartbeat_at": &now,
		"wal_bytes": payload.Metrics.WALBytes, "input_eps": inputEPS, "output_eps": outputEPS,
		"dropped_total": payload.Metrics.DroppedRecords, "retry_total": payload.Metrics.RetryTotal,
		"last_error": firstNonEmpty(payload.Error, payload.Metrics.LastError), "metrics": string(metricsRaw),
	}
	if !payload.Metrics.LastSuccess.IsZero() {
		updates["last_ingest_at"] = &payload.Metrics.LastSuccess
	}
	instance := logmodel.CollectorInstance{InstanceID: identity.InstanceID}
	err := h.db.WithContext(c.Request.Context()).Where("instance_id = ?", identity.InstanceID).Assign(updates).FirstOrCreate(&instance).Error
	return instance, err
}

func syncKubernetesAssignments(tx *gorm.DB, identity kubernetesCollectorIdentity) (logmodel.CollectorInstance, error) {
	now := time.Now()
	instance := logmodel.CollectorInstance{InstanceID: identity.InstanceID}
	if err := tx.Where("instance_id = ?", identity.InstanceID).Assign(map[string]any{
		"agent_id": identity.InstanceID, "mode": "kubernetes-node", "cluster_id": identity.ClusterID,
		"hostname": identity.NodeName, "node_name": identity.NodeName, "collector_type": "opshub-agent",
		"status": "online", "last_heartbeat_at": &now,
	}).FirstOrCreate(&instance).Error; err != nil {
		return instance, err
	}
	var policyIDs []uint
	if err := tx.Model(&logmodel.PolicyTarget{}).Distinct("policy_id").Where("target_type = ? AND target_id = ?", "cluster", identity.ClusterID).Pluck("policy_id", &policyIDs).Error; err != nil {
		return instance, err
	}
	var policies []logmodel.CollectionPolicy
	if len(policyIDs) > 0 {
		if err := tx.Where("id IN ? AND source_mode = ? AND status = ?", policyIDs, "kubernetes", "published").Find(&policies).Error; err != nil {
			return instance, err
		}
	}
	activeIDs := make([]uint, 0, len(policies))
	for _, policy := range policies {
		activeIDs = append(activeIDs, policy.ID)
		assignment := logmodel.CollectorAssignment{
			InstanceID: identity.InstanceID, PolicyID: policy.ID, PolicyVersion: policy.Version,
			DesiredState: "active", ApplyStatus: "pending",
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "instance_id"}, {Name: "policy_id"}},
			DoUpdates: clause.Assignments(map[string]any{"policy_version": policy.Version, "desired_state": "active"}),
		}).Create(&assignment).Error; err != nil {
			return instance, err
		}
	}
	disable := tx.Model(&logmodel.CollectorAssignment{}).Where("instance_id = ?", identity.InstanceID)
	if len(activeIDs) > 0 {
		disable = disable.Where("policy_id NOT IN ?", activeIDs)
	}
	disable = disable.Where("desired_state <> ?", "disabled")
	if err := disable.Updates(map[string]any{"desired_state": "disabled", "apply_status": "pending", "applied_at": nil}).Error; err != nil {
		return instance, err
	}
	return instance, nil
}

func buildKubernetesLogConfig(tx *gorm.DB, c *gin.Context, identity kubernetesCollectorIdentity, instance logmodel.CollectorInstance) (agentLogConfigResponse, error) {
	var assignments []logmodel.CollectorAssignment
	if err := tx.Where("instance_id = ?", identity.InstanceID).Order("policy_id ASC").Find(&assignments).Error; err != nil {
		return agentLogConfigResponse{}, err
	}
	config := logagent.Config{
		GatewayURL: resolveAgentGatewayURL(c), GatewayToken: resolveAgentGatewayToken(c),
		StateDir: "/var/lib/opshub-log-agent", ScanIntervalSeconds: 1, BatchSize: 500,
		FlushIntervalSeconds: 2, MaxWALBytes: 2 * 1024 * 1024 * 1024,
	}
	desired := make([]agentDesiredAssignment, 0, len(assignments))
	var configVersion uint64
	for _, assignment := range assignments {
		desired = append(desired, agentDesiredAssignment{PolicyID: assignment.PolicyID, PolicyVersion: assignment.PolicyVersion, DesiredState: assignment.DesiredState})
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
		targets := make([]policyTargetInput, 0)
		for _, target := range payload.Targets {
			if target.TargetType == "cluster" && target.TargetID == identity.ClusterID {
				targets = append(targets, target)
			}
		}
		if len(targets) == 0 {
			continue
		}
		config.Sources = append(config.Sources, logagent.SourceConfig{
			ID: fmt.Sprintf("policy-%d", assignment.PolicyID), PolicyID: uint64(assignment.PolicyID), PolicyVersion: assignment.PolicyVersion,
			Format: "cri", Paths: payload.Paths, ExcludePaths: payload.ExcludePaths, ReadFrom: payload.ReadFrom,
			Encoding: payload.Encoding, Environment: payload.Environment, Service: payload.Service,
			Stream: payload.Stream, MaxLineBytes: payload.MaxLineBytes, Parser: payload.Parser, Multiline: payload.Multiline,
			Redaction: payload.Redaction, Retention: payload.Retention,
			Kubernetes: &logagent.KubernetesSourceConfig{
				ClusterID: uint64(identity.ClusterID), ClusterName: identity.Cluster.Name,
				Selectors:         policyTargetsToKubernetesSelectors(targets),
				ExcludeNamespaces: []string{"kube-system", "kube-public", "kube-node-lease"},
				LabelAllowlist:    []string{"app.kubernetes.io/name", "app.kubernetes.io/component", "app.kubernetes.io/instance", "app", "k8s-app", "version", "environment"},
			},
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
		if err := config.Validate(); err != nil {
			return agentLogConfigResponse{}, err
		}
	}
	return agentLogConfigResponse{
		ConfigVersion: configVersion, ReloadGeneration: instance.ReloadGeneration,
		PollInterval: 30, LogCollection: config, Assignments: desired,
	}, nil
}

func kubernetesCollectorInstanceID(clusterID uint, nodeName string) string {
	value := fmt.Sprintf("k8s:%d:%s", clusterID, strings.TrimSpace(nodeName))
	if len(value) <= 150 {
		return value
	}
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("k8s:%d:%s", clusterID, hex.EncodeToString(sum[:12]))
}
