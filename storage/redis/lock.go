package redis

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	// lockDefaultTTL 锁默认 TTL
	lockDefaultTTL = 30 * time.Second
	// lockRenewInterval 看门狗续期间隔（TTL 的 1/3）
	lockRenewInterval = 10 * time.Second
)

// DistributedLock 基于 Redis SETNX + Lua 脚本的分布式锁，实现 interfaces.DistributedLock 接口
type DistributedLock struct {
	client *Client
	key    string
	value  string
	ttl    time.Duration
	stopCh chan struct{}
}

// NewDistributedLock 创建分布式锁，key 格式：workflow:lock:instance:{instanceID}:node:{nodeID}
func NewDistributedLock(client *Client, key string, ttl time.Duration) *DistributedLock {
	if ttl <= 0 {
		ttl = lockDefaultTTL
	}
	return &DistributedLock{
		client: client,
		key:    key,
		value:  uuid.NewString(), // 唯一持有者标识，防止误释放
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}
}

// Lock 获取锁并启动看门狗自动续期
func (l *DistributedLock) Lock() (bool, error) {
	ok, err := l.client.Lock(l.key, l.value, l.ttl)
	if err != nil {
		return false, fmt.Errorf("lock key=%s: %w", l.key, err)
	}
	if ok {
		go l.watchdog()
		log.Debug().Str("key", l.key).Msg("distributed lock acquired")
	}
	return ok, nil
}

// Unlock 释放锁并停止看门狗
func (l *DistributedLock) Unlock() (bool, error) {
	// 停止看门狗
	select {
	case <-l.stopCh:
		// 已关闭
	default:
		close(l.stopCh)
	}

	ok, err := l.client.Unlock(l.key, l.value)
	if err != nil {
		return false, fmt.Errorf("unlock key=%s: %w", l.key, err)
	}
	if ok {
		log.Debug().Str("key", l.key).Msg("distributed lock released")
	}
	return ok, nil
}

// TryLockWithTimeout 带超时时间的锁获取，轮询尝试直到成功或超时
func (l *DistributedLock) TryLockWithTimeout(timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	retryInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		ok, err := l.client.Lock(l.key, l.value, l.ttl)
		if err != nil {
			return false, fmt.Errorf("try lock key=%s: %w", l.key, err)
		}
		if ok {
			go l.watchdog()
			log.Debug().Str("key", l.key).Msg("distributed lock acquired (with timeout)")
			return true, nil
		}
		time.Sleep(retryInterval)
	}
	return false, nil
}

// watchdog 看门狗协程，每 lockRenewInterval 自动续期
func (l *DistributedLock) watchdog() {
	ticker := time.NewTicker(lockRenewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-l.stopCh:
			return
		case <-ticker.C:
			ok, err := l.client.RenewLock(l.key, l.value, l.ttl)
			if err != nil {
				log.Warn().Err(err).Str("key", l.key).Msg("watchdog renew lock error")
				return
			}
			if !ok {
				log.Warn().Str("key", l.key).Msg("watchdog renew lock failed: lock not held")
				return
			}
			log.Debug().Str("key", l.key).Msg("watchdog renew lock success")
		}
	}
}

// LockKeyFormat 生成节点锁的 key
func LockKeyFormat(instanceID, nodeID string) string {
	return fmt.Sprintf("workflow:lock:instance:%s:node:%s", instanceID, nodeID)
}
