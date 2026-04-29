package node

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// HTTPExecutor HTTP 请求执行器
// 配置字段：
//   - url     (string): 请求 URL，必填
//   - method  (string): 请求方法，默认 GET
//   - timeout_ms (int): 请求超时毫秒数，默认 5000
//   - headers (map[string]string): 请求头
//   - body    (string): 请求体（POST/PUT 时使用）
type HTTPExecutor struct {
	client *http.Client
}

// NewHTTPExecutor 创建 HTTP 执行器
func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{client: &http.Client{}}
}

func (e *HTTPExecutor) Type() string { return "http" }

func (e *HTTPExecutor) Execute(config map[string]interface{}, input map[string]interface{}, ctx *types.WorkflowContext) (map[string]interface{}, error) {
	urlStr, _ := config["url"].(string)
	if urlStr == "" {
		return nil, fmt.Errorf("http executor: 'url' is required")
	}

	method := "GET"
	if m, ok := config["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	timeoutMs := 5000
	switch v := config["timeout_ms"].(type) {
	case float64:
		timeoutMs = int(v)
	case int:
		timeoutMs = v
	}

	var bodyReader io.Reader
	if bodyStr, ok := config["body"].(string); ok && bodyStr != "" {
		bodyReader = strings.NewReader(bodyStr)
	}

	req, err := http.NewRequest(method, urlStr, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("http executor: create request: %w", err)
	}

	// 设置请求头
	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if sv, ok := v.(string); ok {
				req.Header.Set(k, sv)
			}
		}
	}

	httpClient := e.client
	if timeoutMs > 0 {
		httpClient = &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http executor: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	log.Info().
		Str("instance_id", ctx.InstanceID).
		Str("url", urlStr).
		Int("status_code", resp.StatusCode).
		Msg("http executor completed")

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http executor: server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return map[string]interface{}{
		"status_code":   resp.StatusCode,
		"response_body": string(respBody),
	}, nil
}


