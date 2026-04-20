/*
 * @Date: 2026-04-20 16:22:31
 * @LastEditTime: 2026-04-20 18:09:48
 * @FilePath: /dark_pkg/pkg/natsmgr/nats_msg_queue.go
 * @Description:
 */
package natsmgr

import (
	"time"

	"github.com/nats-io/nats.go"
)

type NatsMsgQueue struct {
	JetStream nats.JetStreamContext
}

// NewMsgQueue 返回一个消息队列管理实例
func NewNatsMsgQueue(jsNats nats.JetStreamContext) *NatsMsgQueue {
	return &NatsMsgQueue{
		JetStream: jsNats,
	}
}

// --------------------- 普通消息 ---------------------

func (m *NatsMsgQueue) Publish(subject string, data []byte) error {
	_, err := m.JetStream.Publish(subject, data)
	return err
}

// --------------------- 延迟消息 ---------------------

func (m *NatsMsgQueue) PublishDelayed(subject string, data []byte, delay time.Duration) error {
	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  nats.Header{},
	}
	msg.Header.Set("Nats-Delay", delay.String())
	_, err := m.JetStream.PublishMsg(msg)
	return err
}

// --------------------- 去重消息 ---------------------

func (m *NatsMsgQueue) PublishDedup(subject string, data []byte, msgID string) error {
	_, err := m.JetStream.Publish(subject, data, nats.MsgId(msgID))
	return err
}

// --------------------- 广播消息 ---------------------

func (m *NatsMsgQueue) PublishBroadcast(subject string, data []byte) error {
	return m.Publish(subject, data)
}
