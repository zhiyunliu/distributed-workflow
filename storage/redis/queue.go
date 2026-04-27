package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/zhiyunliu/distributed-workflow/types"
)

const (
	// dequeueTimeout BRPOP 阻塞等待超时，单位秒
	dequeueTimeout = 3 * time.Second
)

// TaskQueue 基于 Redis List 的任务队列，实现 TaskQueueManager 接口
type TaskQueue struct {
	client *Client
	key    string
	stopCh chan struct{}
}

// NewTaskQueue 创建任务队列，key 为 Redis List 的键名
func NewTaskQueue(client *Client, key string) *TaskQueue {
	if key == "" {
		key = TaskQueueKey
	}
	return &TaskQueue{
		client: client,
		key:    key,
		stopCh: make(chan struct{}),
	}
}

// Enqueue 任务入队（LPUSH）
func (q *TaskQueue) Enqueue(task *types.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	if err := q.client.EnqueueTask(q.key, string(data)); err != nil {
		return fmt.Errorf("enqueue task instance=%s node=%s: %w",
			task.InstanceID, task.NodeID, err)
	}
	log.Debug().
		Str("task_id", task.TaskID).
		Str("instance_id", task.InstanceID).
		Str("node_id", task.NodeID).
		Msg("task enqueued")
	return nil
}

// Dequeue 任务出队（BRPOP，阻塞等待）
// 调用者应循环调用本方法，返回 nil task 表示超时无数据，非 nil error 表示异常
func (q *TaskQueue) Dequeue() (*types.Task, error) {
	data, err := q.client.DequeueTask(q.key, dequeueTimeout)
	if err != nil {
		return nil, fmt.Errorf("dequeue task: %w", err)
	}
	if data == "" {
		return nil, nil // 超时，无数据
	}
	var task types.Task
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}
	log.Debug().
		Str("task_id", task.TaskID).
		Str("instance_id", task.InstanceID).
		Str("node_id", task.NodeID).
		Msg("task dequeued")
	return &task, nil
}

// Remove 从队列移除指定任务（LREM），使用 O(n) 扫描
// 适合低频的任务取消场景
func (q *TaskQueue) Remove(instanceID string, nodeID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRedisTimeout)
	defer cancel()

	// 获取队列所有元素进行匹配（D1 阶段队列规模可控）
	items, err := q.client.rdb.LRange(ctx, q.key, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("lrange for remove: %w", err)
	}

	for _, item := range items {
		var task types.Task
		if jsonErr := json.Unmarshal([]byte(item), &task); jsonErr != nil {
			continue
		}
		if task.InstanceID == instanceID && task.NodeID == nodeID {
			rmCtx, rmCancel := context.WithTimeout(context.Background(), defaultRedisTimeout)
			_, err = q.client.rdb.LRem(rmCtx, q.key, 1, item).Result()
			rmCancel()
			if err != nil {
				return fmt.Errorf("lrem task instance=%s node=%s: %w", instanceID, nodeID, err)
			}
			log.Debug().Str("instance_id", instanceID).Str("node_id", nodeID).Msg("task removed from queue")
			return nil
		}
	}
	return nil // 不存在不报错
}

// Len 获取队列长度
func (q *TaskQueue) Len() int {
	n, err := q.client.GetQueueLength(q.key)
	if err != nil {
		log.Warn().Err(err).Msg("get queue length failed")
		return 0
	}
	return int(n)
}
