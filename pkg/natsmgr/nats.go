/*
 * @Date: 2026-04-20 15:22:24
 * @LastEditTime: 2026-04-20 18:09:52
 * @FilePath: /dark_pkg/pkg/natsmgr/nats.go
 * @Description:
 */
package natsmgr

import (
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/nats-io/nats.go"
	"github.com/ye-f-ying/dark_pkg/pkg/config"
)

// nats初步配置
var (
	natsConn *nats.Conn
	js       nats.JetStreamContext
	once     sync.Once
	natsCfg  *config.NATSConfig
)

func GetNATSConfig() *config.NATSConfig {
	if natsCfg == nil {
		cfg := config.GetNATSConfig()
		if cfg == nil {
			cfg = &config.NATSConfig{
				Endpoints:     []string{},
				Username:      "",
				Password:      "",
				ReconnectWait: 1,
				MaxReconnects: -1,
			}
		}
		natsCfg = cfg
	}
	return natsCfg
}

func GetNATSConn() *nats.Conn {
	once.Do(func() {
		cfg := GetNATSConfig()
		if len(cfg.Endpoints) == 0 {
			hlog.Errorf("nats 尚未配置！")
			panic("NATS is not yet configured")
		}
		serviceID := config.GetServerID()
		urls := strings.Join(cfg.Endpoints, ",")
		var err error
		natsConn, err = nats.Connect(urls,
			nats.Name("server_"+serviceID),
			nats.UserInfo(cfg.Username, cfg.Password),
			nats.MaxReconnects(cfg.MaxReconnects),
			nats.ReconnectWait(time.Second*time.Duration(cfg.ReconnectWait)),
		)
		if err != nil {
			hlog.Errorf("nats 连接异常：%v", err)
			panic(err)
		}

		// JetStream 初始化
		js, err = natsConn.JetStream()
		if err != nil {
			hlog.Errorf("JetStream 初始化失败：%v", err)
			panic(err)
		}
	})
	return natsConn
}

/**
 * @description: nats JetStream
 * @return {*}
 */
func GetNATSJetStream() nats.JetStreamContext {
	if js == nil {
		GetNATSConn()
	}
	return js
}
