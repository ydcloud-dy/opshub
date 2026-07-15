package logagent

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

type Agent struct {
	config  Config
	metrics *Metrics
	wal     *WAL
	tailer  *Tailer
	sender  *Sender
}

type AgentOptions struct {
	KubernetesMetadataResolver KubernetesMetadataResolver
}

func NewAgent(config Config, identity AgentIdentity) (*Agent, error) {
	return NewAgentWithOptions(config, identity, AgentOptions{})
}

func NewAgentWithOptions(config Config, identity AgentIdentity, options AgentOptions) (*Agent, error) {
	config.Normalize()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if identity.AgentID == "" || identity.AssetType == "" || identity.AssetID == 0 {
		return nil, fmt.Errorf("日志采集缺少 Agent 或资产身份")
	}
	for _, source := range config.Sources {
		if source.Kubernetes != nil && options.KubernetesMetadataResolver == nil {
			return nil, fmt.Errorf("Kubernetes 日志源 %s 缺少元数据解析器", source.ID)
		}
	}
	metrics := &Metrics{}
	for _, source := range config.Sources {
		if source.PolicyVersion > metrics.configVersion.Load() {
			metrics.configVersion.Store(source.PolicyVersion)
		}
	}
	wal, err := OpenWAL(filepath.Join(config.StateDir, "wal"), config.MaxWALBytes, config.BatchSize, metrics)
	if err != nil {
		return nil, err
	}
	checkpoints, err := NewCheckpointStore(filepath.Join(config.StateDir, "checkpoints.json"))
	if err != nil {
		_ = wal.Close(false)
		return nil, err
	}
	tailer, err := NewTailerWithOptions(config, checkpoints, wal, metrics, TailerOptions{
		KubernetesMetadataResolver: options.KubernetesMetadataResolver,
	})
	if err != nil {
		_ = wal.Close(false)
		return nil, err
	}
	return &Agent{
		config: config, metrics: metrics, wal: wal, tailer: tailer,
		sender: NewSender(config.GatewayURL, config.GatewayToken, identity, metrics),
	}, nil
}

func (agent *Agent) Metrics() *Metrics {
	return agent.metrics
}

func (agent *Agent) Run(ctx context.Context) error {
	if !agent.config.Enabled {
		return nil
	}
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	go func() {
		defer waitGroup.Done()
		agent.runWALFlusher(ctx)
	}()
	go func() {
		defer waitGroup.Done()
		agent.runSender(ctx)
	}()
	err := agent.tailer.Run(ctx)
	_ = agent.wal.Rotate()
	waitGroup.Wait()
	_ = agent.wal.Close(true)
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func (agent *Agent) runWALFlusher(ctx context.Context) {
	ticker := time.NewTicker(agent.config.FlushInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = agent.wal.Rotate()
			return
		case <-ticker.C:
			if err := agent.wal.Rotate(); err != nil {
				agent.metrics.recordError(err)
			}
		}
	}
}

func (agent *Agent) runSender(ctx context.Context) {
	delay := 500 * time.Millisecond
	for {
		if err := agent.sender.ProcessReady(ctx, agent.wal); err != nil && !errors.Is(err, context.Canceled) {
			agent.metrics.recordError(err)
			delay *= 2
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
		} else {
			delay = 500 * time.Millisecond
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
