/*
 * @Date: 2026-04-20 16:27:21
 * @LastEditTime: 2026-04-20 18:22:27
 * @FilePath: /WSLTest/pkg/messagemgr/message.go
 * @Description:
 */
package messagemgr

import (
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
)

// ---------------------------
// 消息定义
// ---------------------------
type Message struct {
	BizType string        // 业务类型
	Data    []byte        // 消息内容
	MsgID   string        // 消息唯一ID
	Delay   time.Duration // 延迟投递时间

	AckFunc    func() error // 内部 Ack 函数
	AutoAck    bool         // 是否默认自动确认
	RetryFunc  func(delay time.Duration) error
	retryCount uint64 // 已经执行的次数
	retryMax   uint64 // 最大重试次数
	metadata   *nats.MsgMetadata
}

// Ack 手动确认消息
func (m *Message) Ack() error {
	if m.AckFunc != nil {
		return m.AckFunc()
	}
	return nil
}

// Retry 延迟重投消息
func (m *Message) Retry(delay time.Duration) error {
	if m.RetryFunc != nil {
		return m.RetryFunc(delay)
	}
	return nil
}

/**
 * @description: 获取消息
 * @param {any} obj
 * @return {*}
 */
func (m *Message) GetData(obj any) error {
	return json.Unmarshal(m.Data, obj)
}

/**
 * @description: 获取最大重试次数
 * @return {*}
 */
func (m *Message) GetRetryMax() uint64 {
	return m.retryMax
}

/**
 * @description: 获取已经重试的次数
 * @return {*}
 */
func (m *Message) GetRetryCount() uint64 {
	return m.retryCount
}

/**
 * @description: 获取nats元数据
 * @return {*}
 */
func (m *Message) GetMetadata() *nats.MsgMetadata {
	return m.metadata
}

// ---------------------------
// 处理器定义
// ---------------------------
type HandlerOptions struct {
	MaxRetry    int           // 最大重试次数
	AckWait     time.Duration // 消息等待确认时间
	Concurrency int           // 并发数 设置多少条消息等待中防止消息洪峰
}

type HandlerFunc func(msg *Message) error

type Handler struct {
	Transport TransportType  // Transport 类型：queue/broadcast/pubsub
	BizType   string         // 业务类型
	Options   HandlerOptions // 处理器配置
	Handle    HandlerFunc    // 处理函数
}
