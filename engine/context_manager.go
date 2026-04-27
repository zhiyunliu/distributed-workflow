package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zhiyunliu/distributed-workflow/interfaces"
	"github.com/zhiyunliu/distributed-workflow/types"
)

// contextManagerImpl ContextManager 实现（支持 Redis 快照）
type contextManagerImpl struct {
	redis *redis.Client
}

// NewContextManager 创建上下文管理器
func NewContextManager(redisClient *redis.Client) interfaces.ContextManager {
	return &contextManagerImpl{redis: redisClient}
}

// contextKey 构造 Redis Key
func contextKey(instanceID string) string {
	return "workflow:context:" + instanceID
}

// snapshotPrefixKey 构造快照前缀 Redis Key
func snapshotPrefixKey(instanceID string) string {
	return fmt.Sprintf("workflow:context:snapshot:%s:", instanceID)
}

// snapshotFullKey 构造带时间戳的快照 Redis Key（同时用作 snapshotID）
func snapshotFullKey(instanceID string) string {
	return fmt.Sprintf("workflow:context:snapshot:%s:%d", instanceID, time.Now().UnixNano())
}

// CreateContext 创建流程上下文
func (c *contextManagerImpl) CreateContext(instanceID string, inputData map[string]interface{}) (*types.WorkflowContext, error) {
	now := time.Now()
	ctx := &types.WorkflowContext{
		InstanceID: instanceID,
		Data:       inputData,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if ctx.Data == nil {
		ctx.Data = make(map[string]interface{})
	}
	b, err := json.Marshal(ctx)
	if err != nil {
		return nil, fmt.Errorf("marshal context: %w", err)
	}
	bg := context.Background()
	if err := c.redis.Set(bg, contextKey(instanceID), b, 72*time.Hour).Err(); err != nil {
		return nil, fmt.Errorf("redis set context: %w", err)
	}
	return ctx, nil
}

// GetContext 获取流程上下文
func (c *contextManagerImpl) GetContext(instanceID string) (*types.WorkflowContext, error) {
	bg := context.Background()
	b, err := c.redis.Get(bg, contextKey(instanceID)).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("context not found for instance '%s'", instanceID)
	}
	if err != nil {
		return nil, fmt.Errorf("redis get context: %w", err)
	}
	var ctx types.WorkflowContext
	if err := json.Unmarshal(b, &ctx); err != nil {
		return nil, fmt.Errorf("unmarshal context: %w", err)
	}
	return &ctx, nil
}

// UpdateContext 合并更新流程上下文
func (c *contextManagerImpl) UpdateContext(instanceID string, data map[string]interface{}) error {
	ctx, err := c.GetContext(instanceID)
	if err != nil {
		return err
	}
	for k, v := range data {
		ctx.Data[k] = v
	}
	ctx.UpdatedAt = time.Now()
	b, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}
	bg := context.Background()
	return c.redis.Set(bg, contextKey(instanceID), b, 72*time.Hour).Err()
}

// DeleteContext 删除上下文
func (c *contextManagerImpl) DeleteContext(instanceID string) error {
	bg := context.Background()
	return c.redis.Del(bg, contextKey(instanceID)).Err()
}

// SaveSnapshot 保存上下文快照，返回快照 ID（用于后续恢复）
func (c *contextManagerImpl) SaveSnapshot(instanceID string) (string, error) {
	bg := context.Background()
	b, err := c.redis.Get(bg, contextKey(instanceID)).Bytes()
	if err == redis.Nil {
		return "", fmt.Errorf("context not found for instance '%s'", instanceID)
	}
	if err != nil {
		return "", fmt.Errorf("get context for snapshot: %w", err)
	}
	key := snapshotFullKey(instanceID)
	// 快照保留 24 小时
	if err := c.redis.Set(bg, key, b, 24*time.Hour).Err(); err != nil {
		return "", fmt.Errorf("save snapshot: %w", err)
	}
	return key, nil
}

// RestoreSnapshot 从指定快照 ID 恢复上下文
func (c *contextManagerImpl) RestoreSnapshot(instanceID string, snapshotID string) error {
	bg := context.Background()
	b, err := c.redis.Get(bg, snapshotID).Bytes()
	if err == redis.Nil {
		return fmt.Errorf("snapshot '%s' not found", snapshotID)
	}
	if err != nil {
		return fmt.Errorf("get snapshot: %w", err)
	}
	return c.redis.Set(bg, contextKey(instanceID), b, 72*time.Hour).Err()
}
