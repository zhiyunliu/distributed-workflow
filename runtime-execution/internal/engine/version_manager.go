package engine

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/internal/common/types"
)

// versionManagerImpl WorkflowVersionManager 实现
type versionManagerImpl struct {
	repo api.WorkflowRepository
}

// NewVersionManager 创建版本管理器
func NewVersionManager(repo api.WorkflowRepository) api.WorkflowVersionManager {
	return &versionManagerImpl{repo: repo}
}

// CreateVersion 基于工作流定义创建新版本（快照式，不覆盖旧版本）
func (v *versionManagerImpl) CreateVersion(workflowID string, def *types.WorkflowDef, changeLog string, createdBy string) (*types.WorkflowVersion, error) {
	// 获取当前最大版本号
	versions, err := v.repo.ListWorkflowVersions(workflowID)
	if err != nil {
		return nil, fmt.Errorf("list workflow versions: %w", err)
	}

	nextVer := 1
	for _, ver := range versions {
		if ver.Version >= nextVer {
			nextVer = ver.Version + 1
		}
	}

	newVer := &types.WorkflowVersion{
		WorkflowID: workflowID,
		Version:    nextVer,
		Definition: def,
		ChangeLog:  changeLog,
		CreatedBy:  createdBy,
		CreatedAt:  time.Now(),
		IsCurrent:  true,
	}

	// 事务：创建版本 + 设置为 current
	if err := v.repo.CreateWorkflowVersion(newVer); err != nil {
		return nil, fmt.Errorf("create workflow version: %w", err)
	}
	if err := v.repo.SetCurrentVersion(workflowID, nextVer); err != nil {
		return nil, fmt.Errorf("set current version: %w", err)
	}
	return newVer, nil
}

// GetVersion 获取指定版本定义
func (v *versionManagerImpl) GetVersion(workflowID string, version int) (*types.WorkflowVersion, error) {
	return v.repo.GetWorkflowVersion(workflowID, version)
}

// GetCurrentVersion 获取当前生效版本
func (v *versionManagerImpl) GetCurrentVersion(workflowID string) (*types.WorkflowVersion, error) {
	return v.repo.GetCurrentWorkflowVersion(workflowID)
}

// ListVersions 列举所有版本
func (v *versionManagerImpl) ListVersions(workflowID string) ([]*types.WorkflowVersion, error) {
	return v.repo.ListWorkflowVersions(workflowID)
}

// SetCurrentVersion 设置当前生效版本
func (v *versionManagerImpl) SetCurrentVersion(workflowID string, version int) error {
	return v.repo.SetCurrentVersion(workflowID, version)
}

// SelectVersionForInstance 为新实例选择版本（考虑灰度配置）
// tenantID 可用于白名单灰度匹配
func (v *versionManagerImpl) SelectVersionForInstance(workflowID string, tenantID string) (int, error) {
	versions, err := v.repo.ListWorkflowVersions(workflowID)
	if err != nil {
		return 0, fmt.Errorf("list versions: %w", err)
	}
	if len(versions) == 0 {
		return 0, fmt.Errorf("no versions found for workflow '%s'", workflowID)
	}

	// 找到 current 版本
	var currentVer *types.WorkflowVersion
	var grayVers []*types.WorkflowVersion
	for _, ver := range versions {
		if ver.IsCurrent {
			currentVer = ver
		}
		if ver.GrayConfig != nil && ver.GrayConfig.Enabled {
			grayVers = append(grayVers, ver)
		}
	}
	if currentVer == nil {
		// fallback: 取最大版本号
		currentVer = versions[len(versions)-1]
	}

	// 灰度命中检查（优先级最高的灰度版本）
	for _, gv := range grayVers {
		if matchGrayRelease(gv.GrayConfig, tenantID) {
			return gv.Version, nil
		}
	}
	return currentVer.Version, nil
}

// matchGrayRelease 判断是否命中灰度规则
func matchGrayRelease(cfg *types.GrayReleaseConfig, tenantID string) bool {
	if cfg == nil || !cfg.Enabled {
		return false
	}
	switch cfg.Type {
	case types.GrayReleaseByWhitelist:
		for _, w := range cfg.Whitelist {
			if w == tenantID {
				return true
			}
		}
		return false
	case types.GrayReleaseByRatio:
		if cfg.Ratio <= 0 {
			return false
		}
		if cfg.Ratio >= 100 {
			return true
		}
		// 按比例随机命中
		return rand.Intn(100) < cfg.Ratio
	}
	return false
}

// calcRetryInterval 按策略计算下次重试间隔（秒）
func calcRetryInterval(policy *types.RetryPolicy, retryCount int) int {
	if policy == nil {
		return 30 // default
	}
	switch policy.Strategy {
	case types.RetryStrategyFixed:
		if policy.RetryInterval <= 0 {
			return 30
		}
		return policy.RetryInterval
	case types.RetryStrategyExponential:
		multiplier := policy.Multiplier
		if multiplier <= 0 {
			multiplier = 2.0
		}
		base := policy.RetryInterval
		if base <= 0 {
			base = 5
		}
		interval := int(float64(base) * math.Pow(multiplier, float64(retryCount)))
		maxInterval := policy.MaxInterval
		if maxInterval > 0 && interval > maxInterval {
			interval = maxInterval
		}
		return interval
	default:
		return 30
	}
}
