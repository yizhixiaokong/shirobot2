package core

import "sync"

// Event 基础数据结构定义
type Event struct {
	Type     string                 // 事件类型: message/notice...
	Platform string                 // 来源平台: wechat/slack...
	Data     map[string]interface{} // 原始数据
	Session  *Session               // 会话上下文
}

func (e Event) Reset() {
	e.Type = ""
	e.Platform = ""
	e.Data = nil
	e.Session = nil
}

// Response 响应结构体 (统一响应格式)
type Response struct {
	Type     string            // 响应类型: text/image...
	Data     interface{}       // 平台特定格式数据
	Metadata map[string]string // 元数据
}

// Session 会话上下文 (跨事件状态保持)
type Session struct {
	ID      string
	Values  sync.Map // 并发安全存储
	Expires int64    // 过期时间戳
}

// EventTypes常量
const (
	EventTypeMessage = "message"
	EventTypeNotice  = "notice"
)

// ResponseTypes常量
const (
	ResponseTypeText       = "text"
	ResponseTypeError      = "error"
	ResponseTypeNotHandled = "not_handled"
)
