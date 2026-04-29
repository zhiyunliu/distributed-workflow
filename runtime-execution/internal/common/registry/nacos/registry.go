package nacos

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/rs/zerolog/log"

	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

const (
	// ServiceName NodeWorker 在 Nacos 中注册的服务名
	ServiceName = "distributed-workflow-worker"
	// GroupName Nacos 服务分组
	GroupName = "DEFAULT_GROUP"
	// ClusterName Nacos 集群名
	ClusterName = "DEFAULT"
	// metaCapabilities Worker 能力元数据键名
	metaCapabilities = "capabilities"
	// metaWorkerID Worker ID 元数据键名
	metaWorkerID = "worker_id"
)

// Config Nacos 连接配置
type Config struct {
	// ServerAddr Nacos 服务地址，格式：host:port
	ServerAddr string
	// NamespaceID 命名空间 ID，默认 public
	NamespaceID string
	// LogDir Nacos 客户端日志目录
	LogDir string
	// CacheDir Nacos 缓存目录
	CacheDir string
	// LogLevel 日志级别：debug/info/warn/error，默认 warn
	LogLevel string
}

// Registry Nacos 服务注册与发现客户端
type Registry struct {
	client naming_client.INamingClient
	cfg    Config
}

// NewRegistry 创建 Nacos Registry
func NewRegistry(cfg Config) (*Registry, error) {
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = "localhost:8848"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "warn"
	}
	if cfg.LogDir == "" {
		cfg.LogDir = "/tmp/nacos/log"
	}
	if cfg.CacheDir == "" {
		cfg.CacheDir = "/tmp/nacos/cache"
	}

	host, port, err := parseAddr(cfg.ServerAddr)
	if err != nil {
		return nil, fmt.Errorf("parse nacos server addr '%s': %w", cfg.ServerAddr, err)
	}

	sc := []constant.ServerConfig{
		*constant.NewServerConfig(host, port),
	}
	cc := *constant.NewClientConfig(
		constant.WithNamespaceId(cfg.NamespaceID),
		constant.WithTimeoutMs(5000),
		constant.WithLogDir(cfg.LogDir),
		constant.WithCacheDir(cfg.CacheDir),
		constant.WithLogLevel(cfg.LogLevel),
		constant.WithNotLoadCacheAtStart(true),
	)

	namingClient, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  &cc,
		ServerConfigs: sc,
	})
	if err != nil {
		return nil, fmt.Errorf("create nacos naming client: %w", err)
	}
	log.Info().Str("server", cfg.ServerAddr).Msg("nacos naming client created")
	return &Registry{client: namingClient, cfg: cfg}, nil
}

// Register 向 Nacos 注册 Worker 服务
func (r *Registry) Register(worker *types.NodeWorkerInfo) error {
	capsJSON, err := json.Marshal(worker.Capabilities)
	if err != nil {
		return fmt.Errorf("marshal worker capabilities: %w", err)
	}

	host, port, err := parseAddr(worker.Address)
	if err != nil {
		return fmt.Errorf("parse worker address '%s': %w", worker.Address, err)
	}

	ok, err := r.client.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          host,
		Port:        port,
		ServiceName: ServiceName,
		GroupName:   GroupName,
		ClusterName: ClusterName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true, // 临时实例，断开连接自动下线
		Metadata: map[string]string{
			metaWorkerID:     worker.ID,
			metaCapabilities: string(capsJSON),
		},
	})
	if err != nil {
		return fmt.Errorf("nacos register worker '%s': %w", worker.ID, err)
	}
	if !ok {
		return fmt.Errorf("nacos register worker '%s': returned false", worker.ID)
	}
	log.Info().Str("worker_id", worker.ID).Str("address", worker.Address).Msg("worker registered to nacos")
	return nil
}

// Deregister 向 Nacos 注销 Worker 服务
func (r *Registry) Deregister(worker *types.NodeWorkerInfo) error {
	host, port, err := parseAddr(worker.Address)
	if err != nil {
		return fmt.Errorf("parse worker address '%s': %w", worker.Address, err)
	}

	ok, err := r.client.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          host,
		Port:        port,
		ServiceName: ServiceName,
		GroupName:   GroupName,
		Cluster:     ClusterName,
		Ephemeral:   true,
	})
	if err != nil {
		return fmt.Errorf("nacos deregister worker '%s': %w", worker.ID, err)
	}
	if !ok {
		log.Warn().Str("worker_id", worker.ID).Msg("nacos deregister returned false")
	}
	log.Info().Str("worker_id", worker.ID).Msg("worker deregistered from nacos")
	return nil
}

// GetAllWorkers 从 Nacos 获取所有在线 Worker 实例
func (r *Registry) GetAllWorkers() ([]*types.NodeWorkerInfo, error) {
	instances, err := r.client.SelectInstances(vo.SelectInstancesParam{
		ServiceName: ServiceName,
		GroupName:   GroupName,
		HealthyOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("nacos select instances: %w", err)
	}
	return convertInstances(instances), nil
}

// Subscribe 订阅 Worker 服务变更事件
// callback 会在 Worker 注册/注销时被调用，传入最新的实例列表
func (r *Registry) Subscribe(callback func(workers []*types.NodeWorkerInfo)) error {
	return r.client.Subscribe(&vo.SubscribeParam{
		ServiceName: ServiceName,
		GroupName:   GroupName,
		SubscribeCallback: func(services []model.Instance, err error) {
			if err != nil {
				log.Error().Err(err).Msg("nacos subscribe callback error")
				return
			}
			workers := convertInstances(services)
			callback(workers)
		},
	})
}

// convertInstances 将 Nacos 实例转换为 NodeWorkerInfo 列表
func convertInstances(instances []model.Instance) []*types.NodeWorkerInfo {
	workers := make([]*types.NodeWorkerInfo, 0, len(instances))
	for _, inst := range instances {
		if !inst.Healthy || !inst.Enable {
			continue
		}
		workerID := inst.Metadata[metaWorkerID]
		caps := make(map[string]int)
		if capsJSON, ok := inst.Metadata[metaCapabilities]; ok && capsJSON != "" {
			_ = json.Unmarshal([]byte(capsJSON), &caps)
		}
		addr := fmt.Sprintf("%s:%d", inst.Ip, inst.Port)
		workers = append(workers, &types.NodeWorkerInfo{
			ID:           workerID,
			Address:      addr,
			IP:           inst.Ip,
			Capabilities: caps,
			CurrentLoad:  make(map[string]int),
			Status:       types.WorkerStatusOnline,
		})
	}
	return workers
}

// parseAddr 解析 host:port 字符串
func parseAddr(addr string) (string, uint64, error) {
	var host string
	var portStr string
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			host = addr[:i]
			portStr = addr[i+1:]
			break
		}
	}
	if host == "" || portStr == "" {
		return "", 0, fmt.Errorf("invalid addr format, expected host:port, got: %s", addr)
	}
	port, err := strconv.ParseUint(portStr, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port '%s': %w", portStr, err)
	}
	return host, port, nil
}


