package monitor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ydcloud-dy/opshub/internal/conf"
	monserver "github.com/ydcloud-dy/opshub/plugins/monitor/server"
)

const (
	monitorLeaderKey           = "opshub:monitor:scheduler:leader"
	monitorLeaderTTL           = 15 * time.Second
	monitorLeaderCheckInterval = 2 * time.Second
)

type monitorLeaderElector struct {
	client   *redis.Client
	ctx      context.Context
	cancel   context.CancelFunc
	instance string
	onLeader func(context.Context)

	mu            sync.Mutex
	leaderCancel  context.CancelFunc
	leaderRunning atomic.Bool
}

func newMonitorLeaderElector(parent context.Context, instance string, onLeader func(context.Context)) (*monitorLeaderElector, error) {
	cfg := conf.Get()
	if cfg == nil {
		return nil, fmt.Errorf("配置未初始化")
	}
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.GetRedisAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConn,
	})
	ctx, cancel := context.WithCancel(parent)
	if err := client.Ping(ctx).Err(); err != nil {
		cancel()
		_ = client.Close()
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}
	return &monitorLeaderElector{
		client:   client,
		ctx:      ctx,
		cancel:   cancel,
		instance: instance,
		onLeader: onLeader,
	}, nil
}

func (e *monitorLeaderElector) Start() {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Printf("monitor leader elector crashed: %v\n", recovered)
		}
	}()

	e.tryAcquire()
	ticker := time.NewTicker(monitorLeaderCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			e.resign()
			return
		case <-ticker.C:
			e.check()
		}
	}
}

func (e *monitorLeaderElector) Stop() {
	e.cancel()
	e.resign()
	_ = e.client.Close()
}

func (e *monitorLeaderElector) tryAcquire() {
	ok, err := e.client.SetNX(e.ctx, monitorLeaderKey, e.instance, monitorLeaderTTL).Result()
	if err != nil {
		monserver.SetMonitorSchedulerLeaderStatus(e.instance, "", false, err)
		fmt.Printf("monitor leader election failed: %v\n", err)
		return
	}
	if ok {
		e.promote()
		return
	}
	leaderID, err := e.client.Get(e.ctx, monitorLeaderKey).Result()
	if err == redis.Nil {
		e.tryAcquire()
		return
	}
	if err != nil {
		e.refreshLeaderStatus(err)
		return
	}
	if leaderID == e.instance {
		e.promote()
		return
	}
	if e.clearStaleLeaderWithoutTTL(leaderID) {
		e.tryAcquire()
		return
	}
	e.refreshLeaderStatus(nil)
}

func (e *monitorLeaderElector) check() {
	if e.leaderRunning.Load() {
		if err := e.renew(); err != nil {
			fmt.Printf("monitor leader renew failed: %v\n", err)
			e.demote()
			e.tryAcquire()
		}
		return
	}
	e.tryAcquire()
}

func (e *monitorLeaderElector) promote() {
	if e.leaderRunning.Load() {
		e.refreshLeaderStatus(nil)
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.leaderRunning.Load() {
		e.refreshLeaderStatus(nil)
		return
	}
	leaderCtx, cancel := context.WithCancel(e.ctx)
	e.leaderCancel = cancel
	e.leaderRunning.Store(true)
	e.refreshLeaderStatus(nil)
	fmt.Printf("monitor scheduler became leader: %s\n", e.instance)
	if e.onLeader != nil {
		go func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					fmt.Printf("monitor scheduler leader callback crashed: %v\n", recovered)
				}
				if leaderCtx.Err() == nil && e.leaderRunning.Load() {
					fmt.Printf("monitor scheduler leader callback exited, releasing leader: %s\n", e.instance)
					e.resign()
				}
			}()
			e.onLeader(leaderCtx)
		}()
	}
}

func (e *monitorLeaderElector) demote() {
	if !e.leaderRunning.Load() {
		e.refreshLeaderStatus(nil)
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.leaderCancel != nil {
		e.leaderCancel()
		e.leaderCancel = nil
	}
	e.leaderRunning.Store(false)
	e.refreshLeaderStatus(nil)
	fmt.Printf("monitor scheduler became follower: %s\n", e.instance)
}

func (e *monitorLeaderElector) renew() error {
	script := redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("pexpire", KEYS[1], ARGV[2])
else
	return 0
end
`)
	result, err := script.Run(e.ctx, e.client, []string{monitorLeaderKey}, e.instance, int64(monitorLeaderTTL/time.Millisecond)).Int()
	if err != nil {
		e.refreshLeaderStatus(err)
		return err
	}
	if result == 0 {
		err := fmt.Errorf("当前实例不再持有监控调度 leader 锁")
		e.refreshLeaderStatus(err)
		return err
	}
	e.refreshLeaderStatus(nil)
	return nil
}

func (e *monitorLeaderElector) clearStaleLeaderWithoutTTL(leaderID string) bool {
	ttl, err := e.client.TTL(e.ctx, monitorLeaderKey).Result()
	if err != nil || ttl != -1 {
		return false
	}
	fmt.Printf("monitor scheduler found stale leader lock without ttl, clearing: %s\n", leaderID)
	deleted, err := e.client.Del(e.ctx, monitorLeaderKey).Result()
	if err != nil {
		e.refreshLeaderStatus(err)
		return false
	}
	return deleted > 0
}

func (e *monitorLeaderElector) resign() {
	e.demote()
	script := redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
else
	return 0
end
`)
	_, _ = script.Run(context.Background(), e.client, []string{monitorLeaderKey}, e.instance).Result()
	e.refreshLeaderStatus(nil)
}

func (e *monitorLeaderElector) refreshLeaderStatus(err error) {
	leaderID, getErr := e.client.Get(e.ctx, monitorLeaderKey).Result()
	if getErr == redis.Nil {
		leaderID = ""
		getErr = nil
	}
	if err == nil {
		err = getErr
	}
	monserver.SetMonitorSchedulerLeaderStatus(e.instance, leaderID, e.leaderRunning.Load(), err)
}
