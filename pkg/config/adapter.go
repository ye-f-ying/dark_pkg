/*
 * @Date: 2026-04-15 14:53:03
 * @LastEditTime: 2026-05-30 17:42:11
 * @FilePath: /dark_pkg/pkg/config/adapter.go
 * @Description:
 */
package config

import (
	"fmt"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/spf13/viper"
)

// ConfigAdapter 泛型配置适配器接口，T=基础配置，U=公共配置
type ConfigAdapter[T IConfig] interface {
	Init() error                            // 初始化配置
	GetConfig() (T, error)                  // 获取分离的基础/公共配置（替代原绑定结构体）
	Watch(callback func(newT T, err error)) // 监听配置变化，回调返回新配置（泛型版）
}

/**
 * @description: 全局初始化配置适配器（单例，业务唯一入口）
 * @return {*}
 */
func Init[T IConfig](opts ...ConfigOption) (ConfigAdapter[T], error) {
	var adapter ConfigAdapter[T]
	var err error

	once.Do(func() {
		// 合并默认选项和传入选项
		optsCopy := *defaultOpts
		for _, opt := range opts {
			opt(&optsCopy)
		}

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
			hlog.Warnf("预加载本地基础配置失败：%v 使用默认配置！", loadErr)
			defaultConfig(cfgViper)
		}
		// 环境变量覆盖：自动映射，点分隔转下划线（如etcd_config.addrs → ETCD_CONFIG_ADDRS）
		OverrideByEnv(cfgViper)

		// 命令参数覆盖
		localFlag(cfgViper)

		// 将viper配置绑定到基础配置结构体，拿到强类型基础配置
		var cfg T
		if bindErr := ViperToStruct(cfgViper, &cfg); bindErr != nil {
			err = fmt.Errorf("基础配置绑定结构体失败：%w", bindErr)
			return
		}
		fmt.Println(cfg)

		if optsCopy.Mode == "" {
			optsCopy.Mode = cfg.GetConfigMode()
		}

		// 根据模式创建对应泛型适配器实例
		switch optsCopy.Mode {
		case ModeLocal:
			adapter = &LocalConfig[T]{
				opts: optsCopy,
				cfg:  cfgViper,
			}
		case ModeEtcd: // 只有 etcd 模式才会执行
			etcdCfgFromBase := cfg.GetEtcdConfig()
			if etcdCfgFromBase != nil {
				if len(optsCopy.EtcdAddrs) == 0 && len(etcdCfgFromBase.Endpoints) > 0 {
					optsCopy.EtcdAddrs = etcdCfgFromBase.Endpoints
				}

				if optsCopy.EtcdCommonKey == "" && etcdCfgFromBase.EtcdCommonKey != "" {
					optsCopy.EtcdCommonKey = etcdCfgFromBase.EtcdCommonKey
				}

				if optsCopy.EtcdTimeout == defaultOpts.EtcdTimeout && etcdCfgFromBase.DialTimeout > 0 {
					optsCopy.EtcdTimeout = etcdCfgFromBase.DialTimeout
				}

				if optsCopy.EtcdUser == "" {
					optsCopy.EtcdUser = etcdCfgFromBase.Username
				}

				if optsCopy.EtcdPwd == "" {
					optsCopy.EtcdPwd = etcdCfgFromBase.Password
				}
			}
			if len(optsCopy.EtcdAddrs) == 0 {
				err = fmt.Errorf("etcd模式下，未通过手动参数或本地基础配置配置etcd地址")
				return
			}
			if optsCopy.EtcdCommonKey == "" {
				err = fmt.Errorf("etcd模式下，未通过手动参数或本地基础配置配置etcd公共配置Key")
				return
			}

			// 执行公共配置写入命令
			if optsCopy.IsWrite {
				err = writeCommonToEtcd(&optsCopy)
				if err != nil {
					return
				}
			}

			adapter = &EtcdConfig[T]{
				opts: optsCopy,
				cfg:  cfgViper,
			}
		default: // 默认走本地配置
			adapter = &LocalConfig[T]{
				opts: optsCopy,
				cfg:  cfgViper,
			}
		}

		// 初始化泛型适配器
		err = adapter.Init()
	})

	// 若初始化失败，返回nil和错误
	if err != nil {
		return nil, err
	}
	// 保存全局实例
	globalAdapter = adapter
	globalConfig, _ = adapter.GetConfig()
	return adapter, nil
}

/**
 * @description:获取全局泛型适配器实例
 * @return {*}
 */
func GetGlobalAdapter[T IConfig]() (ConfigAdapter[T], error) {
	if globalAdapter == nil {
		return nil, fmt.Errorf("全局配置适配器未初始化，请先调用Init[T,U]()")
	}
	// 类型断言为对应泛型适配器
	adapter, ok := globalAdapter.(ConfigAdapter[T])
	if !ok {
		return nil, fmt.Errorf("全局适配器类型不匹配，预期[%T]，实际[%T]", adapter, globalAdapter)
	}
	return adapter, nil
}

/**
 * @description: 通过接口获取数据
 * @return {*}
 */
func GetGlobalConfig() IConfig {
	if globalConfig == nil {
		return nil
	}
	if cfg, ok := globalConfig.(IConfig); ok {
		return cfg
	}
	return nil
}
