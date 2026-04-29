package redis

import internalredis "github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/storage/redis"

// Config 是对外暴露的 Redis 配置。
type Config = internalredis.Config

// Client 是对外暴露的 Redis 客户端类型别名。
type Client = internalredis.Client

// NewClient 创建 Redis 客户端。
func NewClient(cfg Config) (*Client, error) {
	return internalredis.NewClient(cfg)
}
