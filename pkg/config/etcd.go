/*
 * @Date: 2026-04-15 14:44:44
 * @LastEditTime: 2026-04-15 16:57:16
 * @FilePath: /dark_pkg/pkg/config/etcd.go
 * @Description:
 */
package config

import (
	"context"
	"fmt"

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
	opts           ConfigOptions    // 配置选项
	remoteViper    *viper.Viper     // 远程配置
	cfg            *viper.Viper     // 最终配置
	etcdClient     *clientv3.Client // etcd客户端
	globalCallback func(T, error)   // 泛型版配置更新回调
}

/**
 * @description:Init 初始化etcd模式：1.加载基础配置 2.命令+环境覆盖 3.连接etcd 4.读取etcd公共配置 5.合并
 * @return {*}
 */
func (m *EtcdConfig[T]) Init() error {
	// 加载并覆盖基础配置（和本地模式一致）
	if m.cfg == nil {
		m.cfg = viper.New()
		if err := LoadConfigFile(m.cfg, m.opts.ConfigPath); err != nil {
			return err
		}
		// 合并默认选项和传入选项
		optsCopy := *defaultOpts
		// 初始化命令行参数
		initFlag()
		if cliConfigPath != "" {
			optsCopy.ConfigPath = cliConfigPath
		}
		if cliConfigWrite != -1 {
			if cliConfigWrite == 1 {
				optsCopy.IsWrite = true
			} else {
				optsCopy.IsWrite = false
			}
		}
		// 初始化viper，加载本地基础配置文件-命令参数>环境变量>本地配置 覆盖，获取有效配置
		cfgViper := viper.New()
		cfgViper.SetConfigFile(optsCopy.ConfigPath)
		cfgViper.SetConfigType("yaml")
		if loadErr := cfgViper.ReadInConfig(); loadErr != nil {
			/*err = fmt.Errorf("预加载本地基础配置失败：%w", loadErr)
			return*/
			hlog.Warnf("预加载本地基础配置失败：%v 使用默认配置！", loadErr)
			defaultConfig(cfgViper)
		}
		// 环境变量覆盖：自动映射，点分隔转下划线（如etcd_config.addrs → ETCD_CONFIG_ADDRS）
		OverrideByEnv(cfgViper)

		// 命令参数覆盖
		localFlag(cfgViper)
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
	m.remoteViper = etcdV

	// 合并基础配置+etcd公共配置
	err = MergeRemoteToLocalSafely(m.cfg, m.remoteViper, m.opts.protectLocalKeys)
	if err != nil {
		return fmt.Errorf("[pkg/config/etcd.go->Init]合并远程配置失败：%w", err)
	}

	// 开启etcd公共配置热更新监听
	go m.watchEtcd()
	hlog.Infof("etcd配置中心模式初始化成功，监听Key：%s", m.opts.EtcdCommonKey)
	return nil
}

// 获取分离的基础/公共配置
func (m *EtcdConfig[T]) GetConfig() (T, error) {
	var base T
	if err := ViperToStruct(m.cfg, &base); err != nil {
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

				if newEtcdV == nil || m.cfg == nil {
					continue
				}
				m.remoteViper = newEtcdV
				// 合并基础配置+etcd公共配置
				err = MergeRemoteToLocalSafely(m.cfg, newEtcdV, nil)
				if err != nil {
					hlog.Infof("[pkg/config/etcd.go]合并更新后的配置失败：%v", err)
					if m.globalCallback != nil {
						var emptyT T
						m.globalCallback(emptyT, err)
					}
					continue
				}

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
	// 3. 写入etcd指定Key
	if err := EtcdPut(client, opts.EtcdCommonKey, data, opts.EtcdTimeout); err != nil {
		return err
	}
	hlog.Infof("✅ 本地公共配置[%s]已成功写入etcd Key[%s]", opts.ConfigPath, opts.EtcdCommonKey)
	return nil
}
