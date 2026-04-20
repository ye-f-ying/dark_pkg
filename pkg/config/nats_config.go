/*
 * @Date: 2026-04-20 16:07:36
 * @LastEditTime: 2026-04-20 18:34:00
 * @FilePath: /dark_pkg/pkg/config/nats_config.go
 * @Description:
 */
package config

import "github.com/cloudwego/hertz/pkg/common/hlog"

type NATSConfig struct {
	Endpoints     []string `mapstructure:"endpoints"`      // NATS 集群地址
	Username      string   `mapstructure:"username"`       // NATS 用户名
	Password      string   `mapstructure:"password"`       // NATS 密码
	ReconnectWait int      `mapstructure:"reconnect_wait"` // 重连间隔时间
	MaxReconnects int      `mapstructure:"max_reconnects"` // 最大重连次输 -1 为无限
}

func GetNATSConfig() *NATSConfig {
	baseCfg := GetGlobalConfig()
	if baseCfg == nil {
		hlog.Error("config not configured")
		return nil
	}
	return baseCfg.GetNatsConfig()
}
