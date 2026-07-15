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
)

const defaultLogConfigPollInterval = 30 * time.Second

type remoteLogConfig struct {
	ConfigVersion    uint64             `json:"configVersion"`
	ReloadGeneration uint64             `json:"reloadGeneration"`
	PollInterval     int                `json:"pollInterval"`
	LogCollection    logagent.Config    `json:"logCollection"`
	Assignments      []remoteAssignment `json:"assignments"`
}

type remoteAssignment struct {
	PolicyID      uint64 `json:"policyId"`
	PolicyVersion uint64 `json:"policyVersion"`
	DesiredState  string `json:"desiredState"`
}

type logAssignmentStatus struct {
	PolicyID      uint64 `json:"policyId"`
	PolicyVersion uint64 `json:"policyVersion"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
}

type managedLogCollector struct {
	mutex            sync.RWMutex
	client           *http.Client
	configPath       string
	baseConfig       *config
	current          *logagent.Agent
	cancel           context.CancelFunc
	done             chan struct{}
	etag             string
	configVersion    uint64
	reloadGeneration uint64
	activeConfig     logagent.Config
}

func runManagedLogCollector(ctx context.Context, client *http.Client, configPath string, cfg *config) error {
	manager := &managedLogCollector{client: client, configPath: configPath, baseConfig: cfg}
	if cfg.LogMetricsAddress == "" {
		cfg.LogMetricsAddress = "127.0.0.1:19877"
	}
	metricsServer := &http.Server{Addr: cfg.LogMetricsAddress, Handler: manager.metricsHandler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logf("日志采集指标监听于 http://%s/metrics", cfg.LogMetricsAddress)
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logf("日志采集指标服务失败: %v", err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = metricsServer.Shutdown(shutdownCtx)
		manager.stop()
	}()

	if cfg.LogCollection.Enabled {
		if err := manager.apply(ctx, cfg.LogCollection, 0, 0); err != nil {
			logf("本地日志采集配置启动失败，等待平台下发: %v", err)
		}
	}

	pollInterval := defaultLogConfigPollInterval
	for {
		remote, nextETag, changed, err := manager.fetch(ctx)
		if err != nil {
			logf("拉取日志采集策略失败，继续使用上一版本: %v", err)
		} else if changed {
			if remote.PollInterval >= 10 && remote.PollInterval <= 300 {
				pollInterval = time.Duration(remote.PollInterval) * time.Second
			}
			if err := manager.apply(ctx, remote.LogCollection, remote.ConfigVersion, remote.ReloadGeneration); err != nil {
				logf("应用日志采集策略失败: %v", err)
				_ = manager.reportConfigStatus(ctx, remote, "failed", err.Error())
			} else {
				manager.mutex.Lock()
				manager.etag = nextETag
				manager.mutex.Unlock()
				_ = manager.persist(remote.LogCollection)
				_ = manager.reportConfigStatus(ctx, remote, "applied", "")
			}
		}
		if err := manager.reportHeartbeat(ctx); err != nil {
			logf("上报日志采集心跳失败: %v", err)
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

func (manager *managedLogCollector) fetch(ctx context.Context) (remoteLogConfig, string, bool, error) {
	target := manager.baseConfig.ServerURL + "/api/v1/public/agents/log-config?agentId=" + url.QueryEscape(manager.baseConfig.AgentID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return remoteLogConfig{}, "", false, err
	}
	request.Header.Set("Authorization", "Bearer "+manager.baseConfig.AgentToken)
	request.Header.Set("X-OpsHub-Agent-ID", manager.baseConfig.AgentID)
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
		return remoteLogConfig{}, "", false, fmt.Errorf("解析配置响应失败: %w", err)
	}
	if wrapped.Code != 0 && wrapped.Code != 200 {
		return remoteLogConfig{}, "", false, errors.New(wrapped.Message)
	}
	var remote remoteLogConfig
	if err := json.Unmarshal(wrapped.Data, &remote); err != nil {
		return remoteLogConfig{}, "", false, fmt.Errorf("解析日志采集配置失败: %w", err)
	}
	return remote, result.Header.Get("ETag"), true, nil
}

func (manager *managedLogCollector) apply(parent context.Context, next logagent.Config, configVersion, reloadGeneration uint64) error {
	next.Normalize()
	if err := next.Validate(); err != nil {
		return err
	}
	manager.mutex.RLock()
	previous := manager.activeConfig
	previousVersion := manager.configVersion
	previousReloadGeneration := manager.reloadGeneration
	manager.mutex.RUnlock()
	manager.stop()

	var candidate *logagent.Agent
	var err error
	if next.Enabled {
		candidate, err = logagent.NewAgent(next, logagent.AgentIdentity{
			AgentID: manager.baseConfig.AgentID, AssetType: "host", AssetID: uint64(manager.baseConfig.HostID), HostID: uint64(manager.baseConfig.HostID),
		})
		if err != nil {
			manager.restore(parent, previous, previousVersion, previousReloadGeneration)
			return err
		}
	}
	manager.launch(parent, candidate, next, configVersion, reloadGeneration)
	return nil
}

func (manager *managedLogCollector) restore(parent context.Context, previous logagent.Config, configVersion, reloadGeneration uint64) {
	if !previous.Enabled {
		return
	}
	previousAgent, err := logagent.NewAgent(previous, logagent.AgentIdentity{
		AgentID: manager.baseConfig.AgentID, AssetType: "host", AssetID: uint64(manager.baseConfig.HostID), HostID: uint64(manager.baseConfig.HostID),
	})
	if err != nil {
		logf("恢复上一版日志采集策略失败: %v", err)
		return
	}
	manager.launch(parent, previousAgent, previous, configVersion, reloadGeneration)
	logf("已恢复上一版日志采集策略")
}

func (manager *managedLogCollector) launch(parent context.Context, candidate *logagent.Agent, next logagent.Config, configVersion, reloadGeneration uint64) {
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
				logf("日志采集运行失败: %v", runErr)
			}
		}()
		logf("日志采集策略已应用，版本 %d，文件源 %d 个", configVersion, len(next.Sources))
	} else {
		logf("当前主机未启用日志采集策略")
	}
	manager.mutex.Unlock()
}

func (manager *managedLogCollector) stop() {
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
			logf("等待旧日志采集流水线停止超时")
		}
	}
}

func (manager *managedLogCollector) persist(logConfig logagent.Config) error {
	copyConfig := *manager.baseConfig
	copyConfig.LogCollection = logConfig
	return saveConfig(manager.configPath, &copyConfig)
}

func (manager *managedLogCollector) reportConfigStatus(ctx context.Context, remote remoteLogConfig, status, errorMessage string) error {
	assignments := make([]logAssignmentStatus, 0, len(remote.Assignments))
	for _, assignment := range remote.Assignments {
		assignmentStatus := status
		assignmentError := errorMessage
		if assignment.DesiredState == "disabled" {
			assignmentStatus = "disabled"
			assignmentError = ""
		}
		assignments = append(assignments, logAssignmentStatus{
			PolicyID: assignment.PolicyID, PolicyVersion: assignment.PolicyVersion, Status: assignmentStatus, Error: assignmentError,
		})
	}
	payload := map[string]any{
		"agentId": manager.baseConfig.AgentID, "agentToken": manager.baseConfig.AgentToken,
		"configVersion": remote.ConfigVersion, "reloadGeneration": remote.ReloadGeneration,
		"status": status, "error": errorMessage, "assignments": assignments,
	}
	return postJSONWithContext(ctx, manager.client, manager.baseConfig.ServerURL+"/api/v1/public/agents/log-config/status", payload, nil)
}

func (manager *managedLogCollector) reportHeartbeat(ctx context.Context) error {
	manager.mutex.RLock()
	current := manager.current
	configVersion := manager.configVersion
	manager.mutex.RUnlock()
	var snapshot logagent.MetricsSnapshot
	if current != nil {
		snapshot = current.Metrics().Snapshot()
	}
	payload := map[string]any{
		"agentId": manager.baseConfig.AgentID, "agentToken": manager.baseConfig.AgentToken,
		"version": version, "mode": "host", "configVersion": configVersion, "metrics": snapshot,
	}
	return postJSONWithContext(ctx, manager.client, manager.baseConfig.ServerURL+"/api/v1/public/agents/log-heartbeat", payload, nil)
}

func (manager *managedLogCollector) metricsHandler() http.Handler {
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
			_, _ = writer.Write([]byte(`{"status":"ok","collector":"idle"}`))
			return
		}
		writer.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = writer.Write([]byte("opshub_log_agent_config_version 0\n"))
	})
}

func postJSONWithContext(ctx context.Context, client *http.Client, target string, payload, output any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	responseValue, err := client.Do(request)
	if err != nil {
		return err
	}
	defer responseValue.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(responseValue.Body, 2*1024*1024))
	if err != nil {
		return err
	}
	if responseValue.StatusCode < 200 || responseValue.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", responseValue.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var wrapped apiResponse
	if err := json.Unmarshal(responseBody, &wrapped); err != nil {
		return err
	}
	if wrapped.Code != 0 && wrapped.Code != 200 {
		return errors.New(wrapped.Message)
	}
	if output != nil && len(wrapped.Data) > 0 {
		return json.Unmarshal(wrapped.Data, output)
	}
	return nil
}
