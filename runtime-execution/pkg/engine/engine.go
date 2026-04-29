package engine

import (
	"github.com/redis/go-redis/v9"

	internalengine "github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/engine"
	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
)

// Config 是对外暴露的引擎配置。
type Config = internalengine.Config

// Engine 是对外暴露的引擎类型别名。
type Engine = internalengine.Engine

// New 创建引擎实例。
// D6 仓储参数可按需注入，示例场景可直接传 nil。
func New(
	cfg Config,
	repo api.WorkflowRepository,
	redisRepo api.RedisRepository,
	redisClient *redis.Client,
	formRepo api.FormRepository,
	advApprovalRepo api.AdvancedApprovalRepository,
	analyticsRepo api.AnalyticsRepository,
	pluginRepo api.PluginRepository,
	oauthRepo api.OAuthRepository,
) *Engine {
	return internalengine.New(
		cfg,
		repo,
		redisRepo,
		redisClient,
		formRepo,
		advApprovalRepo,
		analyticsRepo,
		pluginRepo,
		oauthRepo,
	)
}
