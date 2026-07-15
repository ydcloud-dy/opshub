//go:build linux

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ydcloud-dy/opshub/internal/logagent"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type kubernetesNodeConfig struct {
	ServerURL      string
	ClusterID      uint
	ClusterToken   string
	NodeName       string
	PodName        string
	Namespace      string
	MetricsAddress string
}

type kubernetesCollectorManager struct {
	mutex            sync.RWMutex
	client           *http.Client
	config           kubernetesNodeConfig
	resolver         logagent.KubernetesMetadataResolver
	instanceID       string
	current          *logagent.Agent
	cancel           context.CancelFunc
	done             chan struct{}
	etag             string
	configVersion    uint64
	reloadGeneration uint64
	activeConfig     logagent.Config
}

func runKubernetesNodeCollector(ctx context.Context, config kubernetesNodeConfig) error {
	config.ServerURL = strings.TrimRight(strings.TrimSpace(config.ServerURL), "/")
	config.ClusterToken = strings.TrimSpace(config.ClusterToken)
	config.NodeName = strings.TrimSpace(config.NodeName)
	if config.ServerURL == "" || config.ClusterID == 0 || config.ClusterToken == "" || config.NodeName == "" {
		return fmt.Errorf("kubernetes-node 模式缺少 server、cluster-id、cluster-token 或 node-name")
	}
	if config.MetricsAddress == "" {
		config.MetricsAddress = "0.0.0.0:19877"
	}
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("读取 Kubernetes 集群内配置失败: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("创建 Kubernetes 客户端失败: %w", err)
	}
	resolver, err := logagent.NewKubernetesResolver(ctx, clientset, config.NodeName)
	if err != nil {
		return err
	}
	manager := &kubernetesCollectorManager{
		client: &http.Client{Timeout: 30 * time.Second}, config: config, resolver: resolver,
		instanceID: fmt.Sprintf("k8s:%d:%s", config.ClusterID, config.NodeName),
	}
	if err := manager.register(ctx); err != nil {
		return err
	}
	metricsServer := &http.Server{Addr: config.MetricsAddress, Handler: manager.metricsHandler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logf("Kubernetes 日志采集指标监听于 http://%s/metrics", config.MetricsAddress)
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logf("Kubernetes 日志采集指标服务失败: %v", err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = metricsServer.Shutdown(shutdownCtx)
		manager.stop()
	}()

	pollInterval := defaultLogConfigPollInterval
	for {
		remote, nextETag, changed, err := manager.fetch(ctx)
		if err != nil {
			logf("拉取 Kubernetes 日志策略失败，继续使用上一版本: %v", err)
		} else if changed {
			if remote.PollInterval >= 10 && remote.PollInterval <= 300 {
				pollInterval = time.Duration(remote.PollInterval) * time.Second
			}
			if err := manager.apply(ctx, remote.LogCollection, remote.ConfigVersion, remote.ReloadGeneration); err != nil {
				logf("应用 Kubernetes 日志策略失败: %v", err)
				_ = manager.reportConfigStatus(ctx, remote, "failed", err.Error())
			} else {
				manager.mutex.Lock()
				manager.etag = nextETag
				manager.mutex.Unlock()
				_ = manager.reportConfigStatus(ctx, remote, "applied", "")
			}
		}
		if err := manager.reportHeartbeat(ctx); err != nil {
			logf("上报 Kubernetes 日志采集心跳失败: %v", err)
		}
		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func (manager *kubernetesCollectorManager) register(ctx context.Context) error {
	payload := manager.basePayload()
	payload["version"] = version
	var result struct {
		InstanceID string `json:"instanceId"`
	}
	if err := postJSONWithContext(ctx, manager.client, manager.config.ServerURL+"/api/v1/public/log-collectors/kubernetes/register", payload, &result); err != nil {
		return fmt.Errorf("注册 Kubernetes 日志采集器失败: %w", err)
	}
	if result.InstanceID != "" {
		manager.instanceID = result.InstanceID
	}
	logf("Kubernetes 日志采集器已注册: cluster=%d node=%s", manager.config.ClusterID, manager.config.NodeName)
	return nil
}

func (manager *kubernetesCollectorManager) fetch(ctx context.Context) (remoteLogConfig, string, bool, error) {
	query := url.Values{}
	query.Set("clusterId", fmt.Sprintf("%d", manager.config.ClusterID))
	query.Set("nodeName", manager.config.NodeName)
	target := manager.config.ServerURL + "/api/v1/public/log-collectors/kubernetes/config?" + query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return remoteLogConfig{}, "", false, err
	}
	request.Header.Set("Authorization", "Bearer "+manager.config.ClusterToken)
	request.Header.Set("X-OpsHub-Cluster-ID", fmt.Sprintf("%d", manager.config.ClusterID))
	request.Header.Set("X-OpsHub-Node-Name", manager.config.NodeName)
	manager.mutex.RLock()
	request.Header.Set("If-None-Match", manager.etag)
	manager.mutex.RUnlock()
	result, err := manager.client.Do(request)
	if err != nil {
		return remoteLogConfig{}, "", false, err
	}
	defer result.Body.Close()
	if result.StatusCode == http.StatusNotModified {
		return remoteLogConfig{}, "", false, nil
	}
	body, err := io.ReadAll(io.LimitReader(result.Body, 4*1024*1024))
	if err != nil {
		return remoteLogConfig{}, "", false, err
	}
	if result.StatusCode < 200 || result.StatusCode >= 300 {
		return remoteLogConfig{}, "", false, fmt.Errorf("HTTP %d: %s", result.StatusCode, strings.TrimSpace(string(body)))
	}
	var wrapped apiResponse
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return remoteLogConfig{}, "", false, err
	}
	if wrapped.Code != 0 && wrapped.Code != 200 {
		return remoteLogConfig{}, "", false, errors.New(wrapped.Message)
	}
	var remote remoteLogConfig
	if err := json.Unmarshal(wrapped.Data, &remote); err != nil {
		return remoteLogConfig{}, "", false, err
	}
	return remote, result.Header.Get("ETag"), true, nil
}

func (manager *kubernetesCollectorManager) apply(parent context.Context, next logagent.Config, configVersion, reloadGeneration uint64) error {
	next.Normalize()
	if err := next.Validate(); err != nil {
		return err
	}
	manager.mutex.RLock()
	previous := manager.activeConfig
	previousVersion := manager.configVersion
	previousReload := manager.reloadGeneration
	manager.mutex.RUnlock()
	manager.stop()
	candidate, err := manager.newAgent(next)
	if err != nil {
		manager.restore(parent, previous, previousVersion, previousReload)
		return err
	}
	manager.launch(parent, candidate, next, configVersion, reloadGeneration)
	return nil
}

func (manager *kubernetesCollectorManager) newAgent(config logagent.Config) (*logagent.Agent, error) {
	if !config.Enabled {
		return nil, nil
	}
	return logagent.NewAgentWithOptions(config, logagent.AgentIdentity{
		AgentID: manager.instanceID, AssetType: "kubernetes", AssetID: uint64(manager.config.ClusterID),
		ClusterID: uint64(manager.config.ClusterID), NodeName: manager.config.NodeName,
	}, logagent.AgentOptions{KubernetesMetadataResolver: manager.resolver})
}

func (manager *kubernetesCollectorManager) restore(parent context.Context, previous logagent.Config, configVersion, reloadGeneration uint64) {
	if !previous.Enabled {
		return
	}
	previousAgent, err := manager.newAgent(previous)
	if err != nil {
		logf("恢复上一版 Kubernetes 日志策略失败: %v", err)
		return
	}
	manager.launch(parent, previousAgent, previous, configVersion, reloadGeneration)
}

func (manager *kubernetesCollectorManager) launch(parent context.Context, candidate *logagent.Agent, next logagent.Config, configVersion, reloadGeneration uint64) {
	manager.mutex.Lock()
	manager.current = candidate
	manager.configVersion = configVersion
	manager.reloadGeneration = reloadGeneration
	manager.activeConfig = next
	if candidate != nil {
		collectorContext, cancel := context.WithCancel(parent)
		manager.cancel = cancel
		manager.done = make(chan struct{})
		done := manager.done
		go func() {
			defer close(done)
			if runErr := candidate.Run(collectorContext); runErr != nil {
				logf("Kubernetes 日志采集运行失败: %v", runErr)
			}
		}()
		logf("Kubernetes 日志策略已应用，版本 %d，日志源 %d 个", configVersion, len(next.Sources))
	}
	manager.mutex.Unlock()
}

func (manager *kubernetesCollectorManager) stop() {
	manager.mutex.Lock()
	cancel, done := manager.cancel, manager.done
	manager.cancel = nil
	manager.done = nil
	manager.current = nil
	manager.mutex.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(15 * time.Second):
			logf("等待旧 Kubernetes 日志流水线停止超时")
		}
	}
}

func (manager *kubernetesCollectorManager) reportConfigStatus(ctx context.Context, remote remoteLogConfig, status, errorMessage string) error {
	assignments := make([]logAssignmentStatus, 0, len(remote.Assignments))
	for _, assignment := range remote.Assignments {
		assignmentStatus := status
		assignmentError := errorMessage
		if assignment.DesiredState == "disabled" {
			assignmentStatus, assignmentError = "disabled", ""
		}
		assignments = append(assignments, logAssignmentStatus{PolicyID: assignment.PolicyID, PolicyVersion: assignment.PolicyVersion, Status: assignmentStatus, Error: assignmentError})
	}
	payload := manager.basePayload()
	payload["configVersion"] = remote.ConfigVersion
	payload["reloadGeneration"] = remote.ReloadGeneration
	payload["status"] = status
	payload["error"] = errorMessage
	payload["assignments"] = assignments
	return postJSONWithContext(ctx, manager.client, manager.config.ServerURL+"/api/v1/public/log-collectors/kubernetes/config/status", payload, nil)
}

func (manager *kubernetesCollectorManager) reportHeartbeat(ctx context.Context) error {
	manager.mutex.RLock()
	current := manager.current
	configVersion := manager.configVersion
	manager.mutex.RUnlock()
	var snapshot logagent.MetricsSnapshot
	if current != nil {
		snapshot = current.Metrics().Snapshot()
	}
	payload := manager.basePayload()
	payload["version"] = version
	payload["configVersion"] = configVersion
	payload["metrics"] = snapshot
	return postJSONWithContext(ctx, manager.client, manager.config.ServerURL+"/api/v1/public/log-collectors/kubernetes/heartbeat", payload, nil)
}

func (manager *kubernetesCollectorManager) basePayload() map[string]any {
	return map[string]any{
		"clusterId": manager.config.ClusterID, "clusterToken": manager.config.ClusterToken,
		"nodeName": manager.config.NodeName, "podName": manager.config.PodName, "namespace": manager.config.Namespace,
	}
}

func (manager *kubernetesCollectorManager) metricsHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		manager.mutex.RLock()
		current := manager.current
		manager.mutex.RUnlock()
		if current != nil {
			current.Metrics().Handler().ServeHTTP(writer, request)
			return
		}
		if request.URL.Path == "/health" {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"status":"ok","collector":"idle","mode":"kubernetes-node"}`))
			return
		}
		writer.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = writer.Write([]byte("opshub_log_agent_config_version 0\n"))
	})
}
