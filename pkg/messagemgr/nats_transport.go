/*
 * @Date: 2026-04-20 16:30:01
 * @LastEditTime: 2026-04-20 18:30:25
 * @FilePath: /dark_pkg/pkg/messagemgr/nats_transport.go
 * @Description:
 */
package messagemgr

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ye-f-ying/dark_pkg/pkg/config"
	"github.com/ye-f-ying/dark_pkg/pkg/natsmgr"
	"github.com/ye-f-ying/dark_pkg/pkg/utils"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/nats-io/nats.go"
)

// ---------------------------
// NATS Transport
// ---------------------------
type NatsTransport struct {
	nc                 *nats.Conn
	js                 nats.JetStreamContext
	lock               sync.Mutex
	streams            map[string]*nats.StreamInfo
	serviceID          string
	handlerConcurrency map[string]int // key: transport|bizType -> allocated concurrency
	subs               map[string]*nats.Subscription
}

// NewNatsTransport 创建 NATS 连接；若 cfg 有 MaxConcurrent 字段会读取（否则使用默认 1000）
func NewNatsTransport() (*NatsTransport, error) {
	cfg := natsmgr.GetNATSConfig()
	serviceId := config.GetServerID()
	if cfg == nil || len(cfg.Endpoints) == 0 {
		hlog.Errorf("nats 尚未配置！")
		return nil, fmt.Errorf("nats is not yet configured")
	}
	urls := cfg.Endpoints[0]
	if len(cfg.Endpoints) > 1 {
		urls = strings.Join(cfg.Endpoints, ",")
	}
	nc, err := nats.Connect(urls,
		nats.Name("server"+serviceId),
		nats.UserInfo(cfg.Username, cfg.Password),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(time.Second*time.Duration(cfg.ReconnectWait)),
	)
	if err != nil {
		hlog.Errorf("nats 连接异常：%v", err)
		return nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		hlog.Errorf("JetStream 初始化失败：%v", err)
		return nil, err
	}

	return &NatsTransport{
		nc:                 nc,
		js:                 js,
		serviceID:          serviceId,
		handlerConcurrency: make(map[string]int),
		subs:               make(map[string]*nats.Subscription),
	}, nil
}

// AddStream 创建 Stream（保持不变）
func (m *NatsTransport) AddStream(name string, subjects []string, storage nats.StorageType) error {
	_, err := m.js.AddStream(&nats.StreamConfig{
		Name:     name,
		Subjects: subjects,
		Storage:  storage,
		Replicas: 1,
	})
	return err
}

// Publish 发布消息
func (m *NatsTransport) Publish(msg *Message, transportType TransportType) error {
	subject := fmt.Sprintf("%s.%s", transportType, msg.BizType)
	nmsg := &nats.Msg{
		Subject: subject,
		Data:    msg.Data,
		Header:  nats.Header{},
	}
	if msg.MsgID != "" {
		nmsg.Header.Set(nats.MsgIdHdr, msg.MsgID)
	}
	if msg.Delay > 0 {
		nmsg.Header.Set("Nats-Delay", fmt.Sprintf("%d", msg.Delay.Milliseconds()))
	}
	if transportType == PubSubTransport {
		err := m.nc.PublishMsg(nmsg)
		return err
	}
	_, err := m.js.PublishMsg(nmsg)
	return err
}

// makeHandlerFunc 注入 Ack / Retry 功能（接收 handler 专属 sem）
// 注：消息处理在回调中同步执行（不再 spawn 新 goroutine），由 sem 控制并发。
//
//	使用 recover 防止 panic 泄漏导致订阅线程退出。
func (m *NatsTransport) makeHandlerFunc(h *Handler) nats.MsgHandler {
	return func(msg *nats.Msg) {

		utils.Go(func() {
			// panic 保护
			defer func() {
				if r := recover(); r != nil {
					hlog.Errorf("[执行消息%s] 出现异常:  %v", h.BizType, r)
				}
			}()
			var retryCount uint64 = 0
			meta, err := msg.Metadata()
			if err == nil && meta != nil {
				retryCount = meta.NumDelivered
			}
			ackCalled := false
			bizMsg := &Message{
				BizType:    h.BizType,
				Data:       msg.Data,
				MsgID:      msg.Header.Get(nats.MsgIdHdr),
				AutoAck:    true,
				retryCount: retryCount,
				retryMax:   uint64(h.Options.MaxRetry),
				metadata:   meta,
				AckFunc: func() error {
					if h.Transport == PubSubTransport { //PubSubTransport 模式不需要ACK
						return nil
					}
					ackCalled = true
					return msg.Ack()
				},
				RetryFunc: func(delay time.Duration) error {
					newMsg := &nats.Msg{
						Subject: msg.Subject,
						Data:    msg.Data,
						Header:  nats.Header{},
					}
					if msg.Header.Get(nats.MsgIdHdr) != "" {
						newMsg.Header.Set(nats.MsgIdHdr, msg.Header.Get(nats.MsgIdHdr))
					}
					if delay > 0 {
						newMsg.Header.Set("Nats-Delay", fmt.Sprintf("%d", int(delay.Milliseconds())))
					}
					if h.Transport == PubSubTransport { //PubSubTransport  用nc
						return m.nc.PublishMsg(newMsg)
					}
					_, err := m.js.PublishMsg(newMsg)
					return err
				},
			}

			// 执行业务
			if err := h.Handle(bizMsg); err != nil {
				if !errors.Is(err, ErrAutomaticHeavyThrow) { // 如果不是自动重投错误则记录错误 并同样重试
					if retryCount < uint64(h.Options.MaxRetry) {
						hlog.Errorf("[执行消息%s] 出现异常: %v  开始重试消息->当前执行次数：%d", h.BizType, err, retryCount)
					} else if retryCount == uint64(h.Options.MaxRetry) {
						hlog.Errorf("[执行消息%s] 出现异常: %v  重试错误已经达到上限:[%d]-当前重试次数：%d", h.BizType, err, h.Options.MaxRetry, retryCount)
					}

				} else { //如果是自动重投错误 记录重投日志
					if retryCount < uint64(h.Options.MaxRetry) {
						hlog.Infof("[消息重投]-[%s]-当前已经执行次数：[%d]", h.BizType, retryCount)
					} else if retryCount == uint64(h.Options.MaxRetry) {
						hlog.Infof("[消息重投次数达到上限]-[%s]-当前已经执行次数：[%d]-上限次数：[%d]", h.BizType, retryCount, h.Options.MaxRetry)
					}
				}
				return
			}

			if !ackCalled && bizMsg.AutoAck && h.Transport != PubSubTransport { //PubSubTransport 不用Ack
				if err := msg.Ack(); err != nil {
					hlog.Errorf("[执行消息%s] 自动 Ack 失败: %v", h.BizType, err)
				}
			}
		})
	}
}

// Close 平滑关闭：先 Drain 所有订阅，再 Drain 连接
func (m *NatsTransport) Close() {
	m.lock.Lock()
	subs := make([]*nats.Subscription, 0, len(m.subs))
	for _, s := range m.subs {
		subs = append(subs, s)
	}
	m.lock.Unlock()

	for _, s := range subs {
		if s == nil {
			continue
		}
		if err := s.Drain(); err != nil {
			hlog.Warnf("subscription drain error: %v", err)
			// 如果 drain 不支持，尝试 unsubscribe
			if err := s.Unsubscribe(); err != nil {
				hlog.Warnf("subscription unsubscribe error: %v", err)
			}
		}
	}

	if m.nc != nil {
		if err := m.nc.Drain(); err != nil {
			hlog.Warnf("nats drain error: %v", err)
		}
		m.nc.Close()
	}
}

/**
 * @description: 投递消息执行处理
 * @param {*Handler} h
 * @return {*}
 */
func (m *NatsTransport) Subscribe(h *Handler) error {
	subject := fmt.Sprintf("%s.%s", h.Transport, h.BizType)
	streamName := "STREAM_" + h.BizType
	key := keyForHandler(h)

	// 保证 Streams map 初始化 & 创建 stream（幂等）
	m.lock.Lock()
	if m.streams == nil {
		m.streams = make(map[string]*nats.StreamInfo)
	}
	allowed, ok := m.handlerConcurrency[key]
	if !ok || allowed <= 0 {
		if h.Options.Concurrency > 0 {
			allowed = h.Options.Concurrency
		} else {
			allowed = 1 //默认一条
		}
	}
	if _, ok := m.streams[h.BizType]; !ok && h.Transport != PubSubTransport {
		stream, err := m.js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{subject},
			Storage:  nats.FileStorage,
		})
		if err != nil && !strings.Contains(err.Error(), "stream name already in use") {
			m.lock.Unlock()
			return err
		}
		m.streams[h.BizType] = stream
	}
	m.lock.Unlock()

	// 在检查 consumer 之前不要创建 sem，确保 sem 的容量与最终 allowed 一致
	// durableName in queue must be cross-instance same
	switch h.Transport {
	case QueueTransport:
		durableName := fmt.Sprintf("%s-durable", h.BizType)
		queueGroup := fmt.Sprintf("queue_%s", h.BizType)

		// 查询 consumerInfo：如果 consumer 已存在，优先 adopt 其 MaxAckPending，或在允许时 UpdateConsumer
		if ci, err := m.js.ConsumerInfo(streamName, durableName); err == nil && ci != nil {
			cfg := ci.Config
			isUpConsumer := false
			//判断设置的与当前释放一致不一致更新
			if cfg.MaxAckPending > 0 && int(cfg.MaxAckPending) != allowed {
				cfg.MaxAckPending = allowed
				isUpConsumer = true
			} else if ci.Config.MaxAckPending > 0 {
				// 一致或非 0：采用已有值（可保证与已存在 consumer 一致）
				allowed = int(ci.Config.MaxAckPending)
			}
			//判断设置的与当前释放一致不一致更新
			if cfg.AckWait != h.Options.AckWait && h.Options.AckWait > 0 {
				isUpConsumer = true
				cfg.AckWait = h.Options.AckWait
			}
			//判断设置的与当前释放一致不一致更新
			if cfg.MaxDeliver != h.Options.MaxRetry && h.Options.MaxRetry > 0 {
				isUpConsumer = true
				cfg.MaxDeliver = h.Options.MaxRetry
			}
			if isUpConsumer {
				//更新Consumer
				if _, uerr := m.js.UpdateConsumer(streamName, &cfg); uerr != nil {
					hlog.Warnf("Subscribe: UpdateConsumer failed for %s: %v; adopting existing MaxAckPending=%d", durableName, uerr, ci.Config.MaxAckPending)
					allowed = int(ci.Config.MaxAckPending)
				} else {
					hlog.Infof("Subscribe: updated consumer %s MaxAckPending -> %d", durableName, allowed)
				}
			}

		} else if err != nil {
			// 如果是 consumer not found，err 可能包含相关信息；不应把它当 fatal 错误
			if !strings.Contains(err.Error(), "consumer not found") {
				hlog.Warnf("Subscribe: ConsumerInfo error stream=%s durable=%s: %v", streamName, durableName, err)
			}
		}

		msgHandler := m.makeHandlerFunc(h)

		sub, err := m.js.QueueSubscribe(subject, queueGroup, msgHandler,
			nats.Durable(durableName),
			nats.ManualAck(),
			nats.AckWait(func() time.Duration {
				if h.Options.AckWait > 0 {
					return h.Options.AckWait
				}
				return 30 * time.Second
			}()),
			nats.MaxDeliver(func() int {
				if h.Options.MaxRetry > 0 {
					return h.Options.MaxRetry
				}
				return 3
			}()),
			nats.MaxAckPending(allowed),
		)
		if err != nil {
			return err
		}
		// 保存 sub 以便 Close 时 drain/unsubscribe
		m.lock.Lock()
		m.subs[key] = sub
		m.lock.Unlock()
		return nil

	case BroadcastTransport:
		// 广播：每实例 durable 含 serviceID（各实例独立）
		durableName := fmt.Sprintf("%s-durable-%s", h.BizType, m.serviceID)

		// 查询 consumerInfo：如果 consumer 已存在，优先 adopt 其 MaxAckPending，或在允许时 UpdateConsumer
		if ci, err := m.js.ConsumerInfo(streamName, durableName); err == nil && ci != nil {
			cfg := ci.Config
			isUpConsumer := false
			//判断设置的与当前释放一致不一致更新
			if cfg.MaxAckPending > 0 && int(cfg.MaxAckPending) != allowed {
				cfg.MaxAckPending = allowed
				isUpConsumer = true
			} else if ci.Config.MaxAckPending > 0 {
				// 一致或非 0：采用已有值（可保证与已存在 consumer 一致）
				allowed = int(ci.Config.MaxAckPending)
			}
			//判断设置的与当前释放一致不一致更新
			if cfg.AckWait != h.Options.AckWait && h.Options.AckWait > 0 {
				isUpConsumer = true
				cfg.AckWait = h.Options.AckWait
			}
			//判断设置的与当前释放一致不一致更新
			if cfg.MaxDeliver != h.Options.MaxRetry && h.Options.MaxRetry > 0 {
				isUpConsumer = true
				cfg.MaxDeliver = h.Options.MaxRetry
			}
			if isUpConsumer {
				//更新Consumer
				if _, uerr := m.js.UpdateConsumer(streamName, &cfg); uerr != nil {
					hlog.Warnf("Subscribe: UpdateConsumer failed for %s: %v; adopting existing MaxAckPending=%d", durableName, uerr, ci.Config.MaxAckPending)
					allowed = int(ci.Config.MaxAckPending)
				} else {
					hlog.Infof("Subscribe: updated consumer %s MaxAckPending -> %d", durableName, allowed)
				}
			}

		} else if err != nil {
			// 如果是 consumer not found，err 可能包含相关信息；不应把它当 fatal 错误
			if !strings.Contains(err.Error(), "consumer not found") {
				hlog.Warnf("Subscribe: ConsumerInfo error stream=%s durable=%s: %v", streamName, durableName, err)
			}
		}

		msgHandler := m.makeHandlerFunc(h)
		sub, err := m.js.Subscribe(subject, msgHandler,
			nats.Durable(durableName),
			nats.ManualAck(),
			nats.AckWait(func() time.Duration {
				if h.Options.AckWait > 0 {
					return h.Options.AckWait
				}
				return 30 * time.Second
			}()),
			nats.MaxDeliver(func() int {
				if h.Options.MaxRetry > 0 {
					return h.Options.MaxRetry
				}
				return 3
			}()),
			nats.MaxAckPending(allowed),
		)
		if err != nil {
			return err
		}
		m.lock.Lock()
		m.subs[key] = sub
		m.lock.Unlock()
		return nil

	case PubSubTransport:
		// 临时订阅（核心 NATS，不是 JetStream）
		msgHandler := m.makeHandlerFunc(h)
		sub, err := m.nc.Subscribe(subject, msgHandler)
		if err != nil {
			return err
		}
		m.lock.Lock()
		m.subs[key] = sub
		m.lock.Unlock()
		return nil

	default:
		return fmt.Errorf("unsupported transport: %s", h.Transport)
	}
}

// Start: 分配并发后订阅
func (m *NatsTransport) Start(hs []*Handler) error {
	for _, h := range hs {
		hlog.Infof("[业务消息加载]-加载%s", h.BizType)
		if err := m.Subscribe(h); err != nil {
			hlog.Infof("[业务消息加载]-加载%s 出现异常！信息：%s", h.BizType, err.Error())
			return err
		}
	}
	hlog.Infof("[业务消息加载]-全部加载完成！")
	return nil
}

// 测试链接
func (m *NatsTransport) Ping() error {
	if err := m.nc.Flush(); err != nil {
		hlog.Errorf("nats 连接失败！信息：%s", err.Error())
		return nil
	}
	if err := m.nc.LastError(); err != nil {
		hlog.Errorf("nats 连接失败！信息：%s", err.Error())
		return nil
	}
	if !m.nc.IsConnected() {
		return fmt.Errorf("NATS 没有处于连接状态")
	}
	return nil
}

// keyForHandler 生成用于 handlerConcurrency/subs 的 key，包含 transport 和 bizType 以避免冲突
func keyForHandler(h *Handler) string {
	return fmt.Sprintf("%s|%s", string(h.Transport), h.BizType)
}
