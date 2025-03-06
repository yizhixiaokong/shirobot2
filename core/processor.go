package core

import (
	"context"
	"log/slog"
	"strings"
)

// 事件处理逻辑
type EventProcessor struct {
	logger  *slog.Logger
	plugins *PluginManager
}

func NewEventProcessor(logger *slog.Logger, plugins *PluginManager) *EventProcessor {
	return &EventProcessor{
		logger:  logger,
		plugins: plugins,
	}
}

func ParseCommand(input string) (args []string) {
	if !strings.HasPrefix(input, "/") {
		return nil
	}
	input = strings.TrimPrefix(input, "/")
	return splitCommand(input)
}

func splitCommand(input string) []string {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}
	return parts
}

func (ep *EventProcessor) Process(ctx context.Context, e *Event) *Response {
	// 解析命令
	ep.logger.Debug("[engine] try to parse command", "text", e.Data["text"])
	args := ParseCommand(e.Data["text"].(string))
	if len(args) == 0 {
		ep.logger.Info("[engine] not a command, not handled")
		return &Response{Type: ResponseTypeNotHandled}
	}

	ep.logger.Debug("[engine] parsed command", "args", args)
	// 查找命令处理器
	cmd, remainingArgs := ep.plugins.registry.Find(args)
	if cmd == nil {
		ep.logger.Info("[engine] command not found")
		return &Response{Type: ResponseTypeError, Data: "command not found"}
	}

	// 执行命令
	c := Context{
		ctx:      ctx,
		Event:    e,
		Response: &Response{},
		// User:     e.User,
		Session: e.Session,
		Command: cmd,
	}

	if err := cmd.Handler(c, remainingArgs); err != nil {
		return &Response{Type: ResponseTypeError, Data: err.Error()}
	}

	return c.Response
}

// 响应分发逻辑
type ResponseDispatcher struct {
	logger   *slog.Logger
	adapters *AdapterManager
}

func NewResponseDispatcher(logger *slog.Logger, adapters *AdapterManager) *ResponseDispatcher {
	return &ResponseDispatcher{
		logger:   logger,
		adapters: adapters,
	}
}

func (rd *ResponseDispatcher) Dispatch(ctx context.Context, resp Response) {
	for _, adapter := range rd.adapters.GetAll() {
		go func(a Adapter) {
			if err := a.SendResponse(ctx, resp); err != nil {
				rd.logger.Error("[engine] send response failed!", "adapter", a.Name(), "error", err)
			}
		}(adapter)
	}
}
