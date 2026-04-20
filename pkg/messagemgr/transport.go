/*
 * @Date: 2026-04-20 16:28:09
 * @LastEditTime: 2026-04-20 16:28:10
 * @FilePath: /WSLTest/pkg/messagemgr/transport.go
 * @Description:
 */
package messagemgr

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrAutomaticHeavyThrow = errors.New("automatic heavy throw")

// TransportType 定义消息类型
type TransportType string

const (
	QueueTransport     TransportType = "queue"     // 队列模式，多服务部署只被一个实例消费
	BroadcastTransport TransportType = "broadcast" // 广播模式，每个服务实例都收到
	PubSubTransport    TransportType = "pubsub"    // 发布-订阅模式，临时订阅，不依赖 Stream/Durable
)

// Transport 抽象接口，支持多种消息中间件
type Transport interface {
	Publish(msg *Message, transportType TransportType) error
	Subscribe(h *Handler) error
	Start([]*Handler) error
	Close()
	Ping() error
}

// Manager 消息管理器，注册 handler 并统一启动/发布
type Manager struct {
	transport Transport
	handlers  []*Handler
}

// NewManager 创建 Manager
func NewManager(transport Transport) *Manager {
	return &Manager{transport: transport}
}

// RegisterHandler 注册消息处理器
func (m *Manager) RegisterHandler(h *Handler) {
	m.handlers = append(m.handlers, h)
}

// Start 启动所有 handler
func (m *Manager) Start() error {
	return m.transport.Start(m.handlers)
}

// Publish 发布消息
/**
 * @description: 发布消息
 * @param {TransportType} transportType 消息的类型
 * @param {string} bizType //执行的业务
 * @param {interface{}} data //数据
 * @param {string} msgID //消息唯一性。如果确定直接收1条消息时设置 否则 在一定时间内同样的id 会被放弃掉 也就是说一定时间内只接受1条
 * @param {time.Duration} delay //是否延迟 延迟的时长 0 立即投递
 * @return {*}
 */
func (m *Manager) Publish(transportType TransportType, bizType string, data interface{}, msgID string, delay time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	msg := &Message{BizType: bizType, Data: jsonData, MsgID: msgID, Delay: delay}
	return m.transport.Publish(msg, transportType)
}

// Close 关闭 Transport
func (m *Manager) Close() {
	m.transport.Close()
}

// 联通性
func (m *Manager) Ping() error {
	return m.transport.Ping()
}
