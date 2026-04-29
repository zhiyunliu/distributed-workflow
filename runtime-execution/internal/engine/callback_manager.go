package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
)

// callbackManagerImpl HTTP 回调管理器
type callbackManagerImpl struct {
	client *http.Client
}

// NewCallbackManager 创建回调管理器
func NewCallbackManager() *callbackManagerImpl {
	return &callbackManagerImpl{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// InvokeCallback 调用节点生命周期回调（同步，带重试）
func (m *callbackManagerImpl) InvokeCallback(url string, data map[string]interface{}, config *types.NodeCallbackConfig) error {
	if url == "" {
		return nil
	}
	timeout := 30 * time.Second
	retryCount := 0
	if config != nil {
		if config.Timeout > 0 {
			timeout = time.Duration(config.Timeout) * time.Second
		}
		retryCount = config.RetryCount
	}

	client := &http.Client{Timeout: timeout}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("callback marshal payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= retryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
		lastErr = m.doPost(client, url, payload)
		if lastErr == nil {
			return nil
		}
		log.Warn().Err(lastErr).Str("url", url).Int("attempt", attempt).Msg("callback invoke failed, retrying")
	}
	return fmt.Errorf("callback invoke failed after %d retries: %w", retryCount, lastErr)
}

// InvokeWebhook 调用 Webhook 通知（异步不重试）
func (m *callbackManagerImpl) InvokeWebhook(url string, event string, data map[string]interface{}) error {
	if url == "" {
		return nil
	}
	payload, err := json.Marshal(map[string]interface{}{
		"event": event,
		"data":  data,
		"time":  time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("webhook marshal payload: %w", err)
	}

	// 异步发送，不阻塞主流程
	go func() {
		if err := m.doPost(m.client, url, payload); err != nil {
			log.Warn().Err(err).Str("url", url).Str("event", event).Msg("webhook invoke failed")
		}
	}()
	return nil
}

func (m *callbackManagerImpl) doPost(client *http.Client, url string, payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "distributed-workflow/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("callback returned non-2xx status: %d", resp.StatusCode)
	}
	return nil
}
