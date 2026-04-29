package node

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
)

// LogExecutor 日志打印执行器，将消息打印到 zerolog
// 配置字段：
//   - message (string): 要打印的消息，支持直接文本
//   - level   (string): 日志级别，可选 info/warn/error，默认 info
type LogExecutor struct{}

func (e *LogExecutor) Type() string { return "log" }

func (e *LogExecutor) Execute(config map[string]interface{}, input map[string]interface{}, ctx *types.WorkflowContext) (map[string]interface{}, error) {
	msg, _ := config["message"].(string)
	if msg == "" {
		msg = "(no message)"
	}
	level, _ := config["level"].(string)

	logEvent := log.Info()
	switch level {
	case "warn":
		logEvent = log.Warn()
	case "error":
		logEvent = log.Error()
	}
	logEvent.
		Str("instance_id", ctx.InstanceID).
		Str("executor", "log").
		Msg(msg)

	return map[string]interface{}{"logged": true, "message": msg}, nil
}

// SleepExecutor 延时执行器，等待指定时间后返回
// 配置字段：
//   - duration_ms (int/float64): 等待毫秒数，默认 0
type SleepExecutor struct{}

func (e *SleepExecutor) Type() string { return "sleep" }

func (e *SleepExecutor) Execute(config map[string]interface{}, input map[string]interface{}, ctx *types.WorkflowContext) (map[string]interface{}, error) {
	var ms int64
	switch v := config["duration_ms"].(type) {
	case float64:
		ms = int64(v)
	case int:
		ms = int64(v)
	case int64:
		ms = v
	}
	if ms < 0 {
		return nil, fmt.Errorf("sleep duration_ms must be >= 0, got %d", ms)
	}
	if ms > 0 {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	return map[string]interface{}{"slept_ms": ms}, nil
}
