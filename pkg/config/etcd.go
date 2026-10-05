/*
 * @Date: 2026-04-15 14:44:44
 * @LastEditTime: 2026-10-05 14:46:07
 * @FilePath: /dark_pkg/pkg/config/etcd.go
 * @Description:
 */
package config

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ETCDConfig struct {
	Address              string   `mapstructure:"address"`                 // 访问地址+注册端口
	Endpoints            []string `mapstructure:"endpoints"`               // etcd 集群地址
	Username             string   `mapstructure:"username"`                // etcd 用户名
	Password             string   `mapstructure:"password"`                // etcd 密码
	DialTimeout          int64    `mapstructure:"dial_timeout"`            // 连接超时时间（秒）
	DialKeepAliveTime    int      `mapstructure:"dial_keep_alive_time"`    // 客户端发起 KeepAlive PING 的周期（秒）
	DialKeepAliveTimeout int      `mapstructure:"dial_keep_alive_timeout"` // 客户端发出 KeepAlive 探测后，等待服务端响应的超时时间
	EtcdCommonKey        string   `mapstructure:"common_key"`              // etcd公共配置Key
}

type OpenTelemetryConfig struct {
	Environment  string  `mapstructure:"environment" json:"environment"`     // 环境（dev/test/prod）
	ExporterAddr string  `mapstructure:"exporter_addr" json:"exporter_addr"` // 导出地址（如otlp:4317）
	SamplerRatio float64 `mapstructure:"sampler_ratio" json:"sampler_ratio"` // 采样率（0-1，1为全采样）
	Enable       bool    `mapstructure:"enable" json:"enable"`               // 是否开启OTel
}

// EtcdConfig Etcd配置中心模式泛型适配器：T=基础配置，U=公共配置
type EtcdConfig[T IConfig] struct {
	opts   ConfigOptions               // 配置选项
	localV *viper.Viper                // 本地基础配置：Init 后视为不可变，任何goroutine不得再Set
	cfg    atomic.Pointer[viper.Viper] // 生效配置：整体原子替换，读无锁
	//cfg            *viper.Viper     // 最终配置
	etcdClient     *clientv3.Client // etcd客户端
	globalCallback func(T, error)   // 泛型版配置更新回调
}

/**
 * @description:Init 初始化etcd模式：1.加载基础配置 2.命令+环境覆盖 3.连接etcd 4.读取etcd公共配置 5.合并
 * @return {*}
 */
func (m *EtcdConfig[T]) Init() error {
	if m.localV == nil { // 如果没有则自动加载本地
		initFlag()
		if cliConfigPath != "" {
			m.opts.ConfigPath = cliConfigPath // 覆盖要写回 m.opts，原来只改了局部 optsCopy
		}
		if cliConfigWrite != -1 {
			m.opts.IsWrite = cliConfigWrite == 1
		}
		v := viper.New()
		if err := LoadConfigFile(v, m.opts.ConfigPath); err != nil {
			return err
		}
		OverrideByEnv(v) // 直接作用在生效配置上，原来作用在被丢弃的 cfgViper 上
		localFlag(v)
		m.localV = v
	}
	// 创建etcd客户端（配置中心模式，etcd不可用则直接失败）
	client, err := NewEtcdClient(&m.opts) // 传指针（工具方法入参为*Options）
	if err != nil {
		return fmt.Errorf("[pkg/config/etcd.go->Init]etcd连接失败（配置中心模式需正常连接）：%w", err)
	}
	m.etcdClient = client

	// 从etcd读取公共配置
	etcdData, err := EtcdGet(client, m.opts.EtcdCommonKey, m.opts.EtcdTimeout)
	if err != nil {
		return fmt.Errorf("[pkg/config/etcd.go->Init]读取etcd公共配置失败：%w", err)
	}

	// etcd配置转Viper
	etcdV, err := BytesToViper(etcdData)
	if err != nil {
		return fmt.Errorf("[pkg/config/etcd.go->Init]解析etcd配置失败：%w", err)
	}

	merged, err := BuildMergedViper(m.localV, etcdV, m.opts.protectLocalKeys)
	if err != nil {
		hlog.Infof("[pkg/config/etcd.go]合并配置失败：%v", err)
		return err
	}
	m.cfg.Store(merged)
	// 开启etcd公共配置热更新监听
	go m.watchEtcd()
	hlog.Infof("etcd配置中心模式初始化成功，监听Key：%s", m.opts.EtcdCommonKey)
	return nil
}

// 获取分离的基础/公共配置
func (m *EtcdConfig[T]) GetConfig() (T, error) {
	var base T
	cfg := m.cfg.Load()
	if cfg == nil {
		return base, fmt.Errorf("[pkg/config/etcd.go->GetConfig]配置未初始化")
	}
	if err := ViperToStruct(cfg, &base); err != nil {
		return base, fmt.Errorf("[pkg/config/etcd.go->GetConfig]绑定基础配置失败：%w", err)
	}
	return base, nil
}

/**
 * @description: watchEtcd 监听etcd公共配置变化，泛型版热更新
 * @return {*}
 */
func (m *EtcdConfig[T]) watchEtcd() {
	// 持续监听etcd Key变化
	watchChan := m.etcdClient.Watch(context.Background(), m.opts.EtcdCommonKey)
	for resp := range watchChan {
		for _, event := range resp.Events {
			if event.Type == clientv3.EventTypePut {
				hlog.Infof("检测到etcd公共配置更新，重新加载并合并...")
				// 解析新的etcd公共配置
				newEtcdV, err := BytesToViper(event.Kv.Value)
				if err != nil {
					hlog.Infof("解析更新后的etcd配置失败：%v", err)
					// 触发回调，返回错误
					if m.globalCallback != nil {
						var emptyT T
						m.globalCallback(emptyT, fmt.Errorf("[pkg/config/etcd.go->watchEtcd]解析etcd新配置失败：%w", err))
					}
					continue
				}

				if newEtcdV == nil || m.cfg.Load() == nil {
					continue
				}
				//m.remoteViper = newEtcdV
				merged, err := BuildMergedViper(m.localV, newEtcdV, m.opts.protectLocalKeys)
				if err != nil {
					hlog.Infof("[pkg/config/etcd.go]合并更新后的配置失败：%v", err)
					if m.globalCallback != nil {
						var emptyT T
						m.globalCallback(emptyT, err)
					}
					continue
				}
				m.cfg.Store(merged)

				// 获取新的配置并触发回调
				newT, err := m.GetConfig()
				if err != nil {
					hlog.Infof("[pkg/config/etcd.go]绑定更新后的配置失败：%v", err)
					if m.globalCallback != nil {
						m.globalCallback(newT, err)
					}
					continue
				}

				// 无错误，触发回调返回新配置
				if m.globalCallback != nil {
					m.globalCallback(newT, nil)
				}
				hlog.Info("etcd公共配置热更新成功，已触发业务回调")
			}
		}
	}
}

/**
 * @description:注册配置更新回调，etcd变化时返回新的基础/公共配置
 * @param {T} newT
 * @param {U} newU
 * @param {error} err
 * @return {*}
 */
func (m *EtcdConfig[T]) Watch(callback func(newT T, err error)) {
	m.globalCallback = callback
}

/**
 * @description: 配置写入etcd逻辑
 * @param {*ConfigOptions} opts
 * @return {*}
 */
func writeCommonToEtcd(opts *ConfigOptions) error {
	// 1. config.yml文件
	data, err := ReadLocalFile(opts.ConfigPath)
	if err != nil {
		return err
	}
	// 2. 创建etcd客户端
	client, err := NewEtcdClient(opts)
	if err != nil {
		return err
	}
	defer client.Close()
	// 3. 写入etcd指定Key -- 后面优化加密后在写入
	if err := EtcdPut(client, opts.EtcdCommonKey, data, opts.EtcdTimeout); err != nil {
		return err
	}
	hlog.Infof("✅ 本地公共配置[%s]已成功写入etcd Key[%s]", opts.ConfigPath, opts.EtcdCommonKey)
	return nil
}
