package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	defaultRedisTimeout = 5 * time.Second
	// 任务队列 key
	TaskQueueKey = "workflow:task:queue"
)

// Config Redis 连接配置
type Config struct {
	// Addr Redis 地址，格式：host:port，默认 localhost:6379
	Addr string
	// Password 密码，无密码时为空
	Password string
	// DB 数据库编号，默认 0
	DB int
	// PoolSize 连接池大小，默认 10
	PoolSize int
}

// Client Redis 客户端封装，实现 RedisRepository 接口
type Client struct {
	rdb *goredis.Client
}

// NewClient 创建 Redis 客户端
func NewClient(cfg Config) (*Client, error) {
	if cfg.Addr == "" {
		cfg.Addr = "localhost:6379"
	}
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 10
	}
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	log.Info().Str("addr", cfg.Addr).Msg("redis connected")
	return &Client{rdb: rdb}, nil
}

// Close 关闭 Redis 连接
func (c *Client) Close() error {
	return c.rdb.Close()
}

// ─────────────────────────────────────────────────────────────────────────────
// RedisRepository 接口实现
// ─────────────────────────────────────────────────────────────────────────────

// Lock 使用 SETNX 获取分布式锁（原子操作）
func (c *Client) Lock(key string, value string, ttl time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRedisTimeout)
	defer cancel()
	result, err := c.rdb.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis lock setnx key=%s: %w", key, err)
	}
	return result, nil
}

// Unlock 使用 Lua 脚本原子释放锁（校验持有者后删除）
func (c *Client) Unlock(key string, value string) (bool, error) {
	const script = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end`
	ctx, cancel := context.WithTimeout(context.Background(), defaultRedisTimeout)
	defer cancel()
	result, err := c.rdb.Eval(ctx, script, []string{key}, value).Int64()
	if err != nil {
		return false, fmt.Errorf("redis unlock eval key=%s: %w", key, err)
	}
	return result == 1, nil
}

// RenewLock 续期锁（看门狗调用）
func (c *Client) RenewLock(key string, value string, ttl time.Duration) (bool, error) {
	const script = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("pexpire", KEYS[1], ARGV[2])
else
    return 0
end`
	ctx, cancel := context.WithTimeout(context.Background(), defaultRedisTimeout)
	defer cancel()
	ms := ttl.Milliseconds()
	result, err := c.rdb.Eval(ctx, script, []string{key}, value, ms).Int64()
	if err != nil {
		return false, fmt.Errorf("redis renew lock eval key=%s: %w", key, err)
	}
	return result == 1, nil
}

// EnqueueTask 任务入队（LPUSH）
func (c *Client) EnqueueTask(key string, task string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRedisTimeout)
	defer cancel()
	if err := c.rdb.LPush(ctx, key, task).Err(); err != nil {
		return fmt.Errorf("redis enqueue lpush key=%s: %w", key, err)
	}
	return nil
}

// DequeueTask 任务出队（BRPOP，阻塞等待）
func (c *Client) DequeueTask(key string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout+time.Second)
	defer cancel()
	result, err := c.rdb.BRPop(ctx, timeout, key).Result()
	if err != nil {
		if err == goredis.Nil {
			return "", nil // 超时无数据
		}
		return "", fmt.Errorf("redis dequeue brpop key=%s: %w", key, err)
	}
	// BRPop 返回 [key, value]
	if len(result) < 2 {
		return "", fmt.Errorf("redis dequeue unexpected result length: %d", len(result))
	}
	return result[1], nil
}

// GetQueueLength 获取队列长度
func (c *Client) GetQueueLength(key string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRedisTimeout)
	defer cancel()
	n, err := c.rdb.LLen(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis llen key=%s: %w", key, err)
	}
	return n, nil
}

// SetCache 设置缓存
func (c *Client) SetCache(key string, value string, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRedisTimeout)
	defer cancel()
	if err := c.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("redis set cache key=%s: %w", key, err)
	}
	return nil
}

// GetCache 获取缓存
func (c *Client) GetCache(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRedisTimeout)
	defer cancel()
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == goredis.Nil {
			return "", nil // key 不存在
		}
		return "", fmt.Errorf("redis get cache key=%s: %w", key, err)
	}
	return val, nil
}

// DeleteCache 删除缓存
func (c *Client) DeleteCache(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRedisTimeout)
	defer cancel()
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del cache key=%s: %w", key, err)
	}
	return nil
}
