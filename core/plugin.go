package core

import (
	"log/slog"
	"sync"
)

// 插件接口 (业务逻辑单元)
type Plugin interface {
	Name() string
	RegisterCommand(registry *CommandRegistry, mws ...Middleware)
}

type PluginManager struct {
	logger   *slog.Logger
	plugins  map[string]Plugin // 插件实例
	registry *CommandRegistry  // 命令注册器
	mu       sync.RWMutex
}

func NewPluginManager(logger *slog.Logger) *PluginManager {
	return &PluginManager{
		logger:   logger,
		plugins:  make(map[string]Plugin),
		registry: NewRegistry(logger),
		mu:       sync.RWMutex{},
	}
}

func (pm *PluginManager) Register(plugin Plugin, mws ...Middleware) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 防止重复注册
	if _, exists := pm.plugins[plugin.Name()]; exists {
		return
	}

	pm.logger.Debug("[plugin] registering plugin", "name", plugin.Name())

	// 注册插件
	pm.plugins[plugin.Name()] = plugin

	// 调用插件的命令注册方法
	plugin.RegisterCommand(pm.registry, mws...)
}
