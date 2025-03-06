package core

import (
	"context"
	"log/slog"
	"sync"
)

// 适配器抽象层
type Adapter interface {
	Name() string
	Start(ctx context.Context, eventChan chan<- Event) error
	SendResponse(ctx context.Context, resp Response) error // 发送响应
}

// 适配器注册管理
type AdapterManager struct {
	logger *slog.Logger

	adapters []Adapter
	mu       sync.RWMutex
}

func NewManager(logger *slog.Logger) *AdapterManager {
	return &AdapterManager{
		logger: logger,

		adapters: make([]Adapter, 0),
		mu:       sync.RWMutex{},
	}
}

func (am *AdapterManager) Register(adapter Adapter) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.adapters = append(am.adapters, adapter)
}

func (am *AdapterManager) GetAll() []Adapter {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.adapters
}
