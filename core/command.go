package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type Command struct {
	Name        string
	Parent      *Command // 父命令
	fullPath    string   // 完整路径
	Aliases     []string // 别名列表
	Description string
	Usage       string
	Handler     CommandHandler

	commands        map[string]*Command // 子命令(可嵌套)
	commandsAliases map[string]string   // 子命令别名
	mu              sync.RWMutex        // 读写锁
}

func (c *Command) AddCommand(cmds ...*Command) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cmd := range cmds {
		cmd.Parent = c
		if c.commands == nil {
			c.commands = make(map[string]*Command)
		}

		c.commands[cmd.Name] = cmd
		for _, alias := range cmd.Aliases {
			if c.commandsAliases == nil {
				c.commandsAliases = make(map[string]string)
			}
			c.commandsAliases[alias] = cmd.Name
		}
	}
}

func (c *Command) SetFullPath() {
	if c.fullPath == "" {
		c.fullPath = c.Name
	}
	for _, subCmd := range c.commands {
		subCmd.fullPath = c.fullPath + " " + subCmd.Name
		subCmd.SetFullPath()
	}
}

func (c *Command) Find(args []string) (*Command, []string) {
	if len(args) == 0 {
		return c, nil
	}

	c.mu.RLock()
	subCmd, exists := c.commands[args[0]]
	c.mu.RUnlock()

	if exists {
		return subCmd.Find(args[1:])
	}

	// 查找别名
	c.mu.RLock()
	name, exists := c.commandsAliases[args[0]]
	c.mu.RUnlock()

	if exists {
		return c.commands[name].Find(args[1:])
	}

	return c, args
}

func (c *Command) GenerateHelp() string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("命令: %s\n", c.fullPath))
	builder.WriteString(fmt.Sprintf("说明: %s\n", c.Description))

	if len(c.commands) > 0 {
		builder.WriteString("子命令:\n")
		for name := range c.commands {
			builder.WriteString(fmt.Sprintf("  %s\n", name))
		}
	}

	return builder.String()
}

type Middleware func(next CommandHandler) CommandHandler
type CommandHandler func(ctx Context, args []string) error

type Context struct {
	ctx      context.Context
	Event    *Event
	Response *Response
	// User     User
	Session *Session
	Command *Command
}
