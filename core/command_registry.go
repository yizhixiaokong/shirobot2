package core

import (
	"log/slog"
	"sync"
)

type CommandRegistry struct {
	logger   *slog.Logger
	commands map[string]*Command
	mu       sync.RWMutex
}

func NewRegistry(logger *slog.Logger) *CommandRegistry {
	return &CommandRegistry{
		logger:   logger,
		commands: make(map[string]*Command),
	}
}

// 注册命令（自动处理别名）
func (r *CommandRegistry) Register(cmd *Command, mws ...Middleware) *Command {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 防止重复注册
	if _, exists := r.commands[cmd.Name]; exists {
		return r.commands[cmd.Name]
	}

	// 为命令添加完整路径
	cmd.SetFullPath()

	r.logger.Debug("[engine] command registry register command", "command", cmd.Name)

	// 注册中间件
	r.logger.Debug("[engine] command registry register middleware", "mws_amount", len(mws), "command", cmd.Name)
	r.registerMiddleware(cmd, mws...)

	for _, subCmd := range cmd.commands {
		r.registerMiddleware(subCmd, mws...)
	}

	// 注册别名
	r.registerCommandWithAlias(cmd)

	r.logger.Debug("[engine] command registry command registered",
		"command", cmd.Name,
		"aliases", cmd.Aliases,
		"description", cmd.Description,
		"usage", cmd.Usage,
		"full_path", cmd.fullPath,
		"sub_commands", len(cmd.commands),
	)

	return cmd
}

func (r *CommandRegistry) registerCommandWithAlias(cmd *Command) {
	r.commands[cmd.Name] = cmd
	for _, alias := range cmd.Aliases {
		r.commands[alias] = cmd
	}
}

// 注册中间件
func (r *CommandRegistry) registerMiddleware(cmd *Command, mws ...Middleware) {
	handler := cmd.Handler
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	cmd.Handler = handler
}

// 查找命令
func (r *CommandRegistry) Find(args []string) (*Command, []string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.logger.Debug("[engine] command registry try to find command", "args", args)

	if len(args) == 0 {
		return nil, nil
	}

	cmdName := args[0]
	cmd, exists := r.commands[cmdName]
	if !exists {
		return nil, args
	}

	currentCmd, remaining := cmd.Find(args[1:])

	r.logger.Debug("[engine] command registry found command",
		"name", currentCmd.Name,
		"aliases", currentCmd.Aliases,
		"description", currentCmd.Description,
		"usage", currentCmd.Usage,
		"full_path", currentCmd.fullPath,
		"args", remaining,
	)

	return currentCmd, remaining
}
