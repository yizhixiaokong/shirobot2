package core

import (
	"context"
	"log/slog"
	"sync"
)

// Engine 核心引擎协调各组件工作
type Engine struct {
	logger *slog.Logger

	adapters   *AdapterManager     // 适配器管理
	plugins    *PluginManager      // 插件管理
	processor  *EventProcessor     // 事件处理器
	dispatcher *ResponseDispatcher // 响应分发器

	eventChan    chan Event    // 事件通道
	responseChan chan Response // 响应通道

	sessionPool sync.Pool   // 会话池
	eventPool   sync.Pool   // 事件池
	workerPool  *WorkerPool // 工作池
}

type EngineConfig struct {
	Logger           *slog.Logger
	EventChanSize    int
	ResponseChanSize int
	WorkerPoolSize   int
}

// EngineOption 引擎配置选项
type EngineOption func(*EngineConfig)

// WithLogger 配置日志记录器
func WithLogger(logger *slog.Logger) EngineOption {
	return func(e *EngineConfig) {
		e.Logger = logger
	}
}

// WithEventChanSize 配置事件通道大小
func WithEventChanSize(size int) EngineOption {
	return func(e *EngineConfig) {
		e.EventChanSize = size
	}
}

// WithResponseChanSize 配置响应通道大小
func WithResponseChanSize(size int) EngineOption {
	return func(e *EngineConfig) {
		e.ResponseChanSize = size
	}
}

// WithWorkerPoolSize 配置工作池大小
func WithWorkerPoolSize(size int) EngineOption {
	return func(e *EngineConfig) {
		e.WorkerPoolSize = size
	}
}

// 全局单例
var (
	_Engine *Engine
	once    sync.Once
)

// NewEngine 创建新引擎实例
func NewEngine(opts ...EngineOption) *Engine {
	once.Do(func() {

		// 默认配置
		ecfg := &EngineConfig{
			Logger:           slog.Default(),
			EventChanSize:    1000, // 默认事件通道大小1000
			ResponseChanSize: 1000, // 默认响应通道大小1000
			WorkerPoolSize:   100,  // 默认工作池大小100
		}

		// 应用配置
		for _, opt := range opts {
			opt(ecfg)
		}

		e := &Engine{}

		e.logger = ecfg.Logger

		e.adapters = NewManager(e.logger)
		e.plugins = NewPluginManager(e.logger)
		e.processor = NewEventProcessor(e.logger, e.plugins)
		e.dispatcher = NewResponseDispatcher(e.logger, e.adapters)

		e.eventChan = make(chan Event, ecfg.EventChanSize)
		e.responseChan = make(chan Response, ecfg.ResponseChanSize)

		e.sessionPool = sync.Pool{
			New: func() interface{} { return &Session{} },
		}
		e.eventPool = sync.Pool{
			New: func() interface{} { return &Event{} },
		}
		e.workerPool = NewWorkerPool(ecfg.WorkerPoolSize)

		e.logger.Debug("[engine] engine created.", "config", ecfg)

		_Engine = e
	})

	return _Engine
}

func GetEngine() *Engine {
	return _Engine
}

func (e *Engine) RegisterAdapter(adapter Adapter) {
	e.adapters.Register(adapter)
}

func (e *Engine) RegisterPlugin(plugin Plugin, mws ...Middleware) {
	e.plugins.Register(plugin, mws...)
}

// Run 启动引擎主循环
func (e *Engine) Run(ctx context.Context) error {
	defer e.cleanup()
	e.logger.Debug("[engine] engine starting...")

	// 启动所有适配器
	for _, adapter := range e.adapters.GetAll() {
		go func(a Adapter) {
			if err := a.Start(ctx, e.eventChan); err != nil {
				e.handleError(err)
			}
			e.logger.Debug("[engine] adapter started.", "adapter", a.Name())
		}(adapter)
	}

	// 启动worker池
	e.workerPool.Start()
	e.logger.Debug("[engine] worker pool started.")

	// 主事件循环
	for {
		select {
		case <-ctx.Done():
			e.logger.Debug("[engine] engine stopping...")
			return ctx.Err()
		case event := <-e.eventChan:
			e.logger.Debug("[engine] engine received event.", "event", event)
			e.workerPool.Submit(func() {
				e.processEvent(ctx, event)
			})
		case resp := <-e.responseChan:
			go e.dispatcher.Dispatch(ctx, resp)
		}
	}
}

// processEvent 处理单个事件
func (e *Engine) processEvent(ctx context.Context, event Event) {
	defer e.recycleEvent(event)

	e.logger.Debug("[engine] engine processing event.", "event", event)
	resp := e.processor.Process(ctx, &event)
	if resp.Type != "" {
		e.responseChan <- *resp
		e.logger.Debug("[engine] engine response sent.", "response", resp)
	}
}

func (e *Engine) recycleEvent(event Event) {
	event.Reset()
	e.eventPool.Put(&event)
}

// cleanup 资源清理
func (e *Engine) cleanup() {
	e.logger.Debug("[engine] engine cleanup...")
	defer e.logger.Debug("[engine] engine cleanup done.")
	close(e.eventChan)
	close(e.responseChan)
	e.workerPool.Stop()

	for _, a := range e.adapters.GetAll() {
		if closer, ok := a.(Closer); ok {
			closer.Close()
		}
	}
}

// handleError 统一错误处理
func (e *Engine) handleError(err error) {
	// 实现错误上报/日志记录
}

// 实现Closer接口用于资源清理
type Closer interface {
	Close()
}

// WorkerPool 协程工作池
type WorkerPool struct {
	workers  int
	taskChan chan func()
	wg       sync.WaitGroup
}

func NewWorkerPool(workers int) *WorkerPool {
	return &WorkerPool{
		workers:  workers,
		taskChan: make(chan func(), 1000),
	}
}

func (wp *WorkerPool) Start() {
	wp.wg.Add(wp.workers)
	for i := 0; i < wp.workers; i++ {
		go func() {
			defer wp.wg.Done()
			for task := range wp.taskChan {
				task()
			}
		}()
	}
}

func (wp *WorkerPool) Submit(task func()) {
	wp.taskChan <- task
}

func (wp *WorkerPool) Stop() {
	close(wp.taskChan)
	wp.wg.Wait()
}
