/*
 * @Date: 2026-04-20 17:06:18
 * @LastEditTime: 2026-04-20 18:31:06
 * @FilePath: /dark_pkg/pkg/messagemgr/messagemgr.go
 * @Description:
 */
package messagemgr

import (
	"sync"

	"github.com/ye-f-ying/dark_pkg/pkg/config"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/nats-io/nats.go"
	"github.com/ye-f-ying/dark_pkg/pkg/natsmgr"
)

var defaultManager *Manager
var defaultManagerOnce sync.Once

/**
 * @description: 默认的任务管理
 * @return {*}
 */
func GetDefaultManager() *Manager {
	defaultManagerOnce.Do(func() {
		cfg := natsmgr.GetNATSConfig()
		if len(cfg.Endpoints) <= 0 {
			hlog.Errorf("[消息/任务管理]-没有配置nats数据！")
			panic("nats 尚未配置！")
		}
		natsTransport := &NatsTransport{
			nc:                 natsmgr.GetNATSConn(),
			js:                 natsmgr.GetNATSJetStream(),
			serviceID:          config.GetServerID(),
			handlerConcurrency: make(map[string]int),
			subs:               make(map[string]*nats.Subscription),
		}
		defaultManager = NewManager(natsTransport)
	})
	return defaultManager
}
