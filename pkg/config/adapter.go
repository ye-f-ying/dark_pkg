/*
 * @Author: yeying
 * @Date: 2026-02-04 13:00:27
 * @FilePath: /dark_pkg/pkg/config/adapter.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/spf13/viper"
)

// 配置模式常量
type ConfigMode string

const (
	ModeLocal   ConfigMode = "local" // 纯本地模式：基础配置+本地common.yml
	ModeEtcd    ConfigMode = "etcd"  // 配置中心模式：基础配置+etcd公共配置
	DefaultMode ConfigMode = ""      // 默认
)

// 配置文件常量（基础/公共分离）
const (
	DefaultAppConfigPath    = "./base.yml"         // 本地基础配置
	DefaultCommonConfigPath = "./config.yml"       // 本地公共配置
	DefaultEtcdCommonKey    = "/config/common/yml" // etcd公共配置Key
	DefaultEtcdTimeout      = 5                    // etcd超时时间(秒)
)

// ConfigAdapter 泛型配置适配器接口，T=基础配置，U=公共配置
type ConfigAdapter[T IBaseConfig, U ICommonConfig] interface {
	Init() error                                    // 初始化配置
	GetConfig() (T, U, error)                       // 获取分离的基础/公共配置（替代原绑定结构体）
	Watch(callback func(newT T, newU U, err error)) // 监听配置变化，回调返回新配置（泛型版）
	SetCmdParams(params map[string]any)             // 设置命令行参数（用于覆盖基础配置）
}

// ConfigOptions 配置适配器选项（扩展IsWrite标记写入命令）
type ConfigOptions struct {
	Mode             ConfigMode // 运行模式：local/etcd
	AppConfigPath    string     // 基础配置文件路径
	CommonConfigPath string     // 本地公共配置文件路径
	EtcdAddrs        []string   // etcd集群地址
	EtcdCommonKey    string     // etcd公共配置Key
	EtcdTimeout      int64      // etcd超时时间
	IsWrite          bool       // 是否是配置写入命令：true则将本地common.yml写入etcd
	EtcdUser         string
	EtcdPwd          string
}

// 定义全局泛型实例的持有者，避免包级泛型（Go不支持包级泛型变量）
var (
	globalAdapter any       // 泛型适配器实例（任意实现ConfigAdapter[T,U]的类型）
	once          sync.Once // 单例初始化锁
	defaultOpts   = &ConfigOptions{
		Mode:             DefaultMode,
		AppConfigPath:    DefaultAppConfigPath,
		CommonConfigPath: DefaultCommonConfigPath,
		EtcdCommonKey:    DefaultEtcdCommonKey,
		EtcdTimeout:      DefaultEtcdTimeout,
		IsWrite:          false,
		EtcdUser:         "",
		EtcdPwd:          "",
	}
)

// Option 选项模式函数
type ConfigOption func(*ConfigOptions)

/**
 * @description:  运行模式：local/etcd
 * @param {ConfigMode} mode
 * @return {*}
 */
func WithMode(mode ConfigMode) ConfigOption {
	return func(o *ConfigOptions) { o.Mode = mode }
}

/**
 * @description:基础配置文件路径
 * @param {string} path
 * @return {*}
 */
func WithAppConfigPath(path string) ConfigOption {
	return func(o *ConfigOptions) { o.AppConfigPath = path }
}

/**
 * @description:通用配置文件路径
 * @param {string} path
 * @return {*}
 */
func WithCommonConfigPath(path string) ConfigOption {
	return func(o *ConfigOptions) { o.CommonConfigPath = path }
}

/**
 * @description:ETCD注册中心地址
 * @param {[]string} addrs
 * @return {*}
 */
func WithEtcdAddrs(addrs []string) ConfigOption {
	return func(o *ConfigOptions) { o.EtcdAddrs = addrs }
}

/**
 * @description: ETCD 账号密码
 * @param {*} user
 * @param {string} pwd
 * @return {*}
 */
func WithEtcdAuth(user, pwd string) ConfigOption {
	return func(o *ConfigOptions) {
		o.EtcdUser = user
		o.EtcdPwd = pwd
	}
}

/**
 * @description:etcd公共配置Key
 * @param {string} key
 * @return {*}
 */
func WithEtcdCommonKey(key string) ConfigOption {
	return func(o *ConfigOptions) { o.EtcdCommonKey = key }
}

/**
 * @description:ETCD超时设置
 * @param {int64} sec
 * @return {*}
 */
func WithEtcdTimeout(sec int64) ConfigOption {
	return func(o *ConfigOptions) { o.EtcdTimeout = sec }
}

/**
 * @description: 是否是配置写入命令：true则将本地common.yml写入etcd
 * @param {bool} isWrite
 * @return {*}
 */
func WithIsWrite(isWrite bool) ConfigOption {
	return func(o *ConfigOptions) { o.IsWrite = isWrite }
}

/**
 * @description: 全局初始化配置适配器（单例，业务唯一入口）
 * @return {*}
 */
func Init[T IBaseConfig, U ICommonConfig](opts ...ConfigOption) (ConfigAdapter[T, U], error) {
	var adapter ConfigAdapter[T, U]
	var err error

	once.Do(func() {
		// 合并默认选项和传入选项
		optsCopy := *defaultOpts
		for _, opt := range opts {
			opt(&optsCopy)
		}
		// 初始化命令行参数
		initFlag()
		if cliBaseConfigPath != "" {
			optsCopy.AppConfigPath = cliBaseConfigPath
		}
		if cliConfigPath != "" {
			optsCopy.CommonConfigPath = cliConfigPath
		}
		if cliConfigWrite != -1 {
			if cliConfigWrite == 1 {
				optsCopy.IsWrite = true
			} else {
				optsCopy.IsWrite = false
			}
		}
		// 初始化viper，加载本地基础配置文件-命令参数>环境变量>本地配置 覆盖，获取有效配置
		baseViper := viper.New()
		baseViper.SetConfigFile(optsCopy.AppConfigPath)
		baseViper.SetConfigType("yaml")
		if loadErr := baseViper.ReadInConfig(); loadErr != nil {
			/*err = fmt.Errorf("预加载本地基础配置失败：%w", loadErr)
			return*/
			hlog.Warnf("预加载本地基础配置失败：%v 使用默认配置！", loadErr)
			defaultConfig(baseViper)
		}
		// 环境变量覆盖：自动映射，点分隔转下划线（如etcd_config.addrs → ETCD_CONFIG_ADDRS）
		baseViper.AutomaticEnv()
		baseViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		// 命令参数覆盖
		localFlag(baseViper)

		// 将viper配置绑定到基础配置结构体，拿到强类型基础配置
		var baseCfg T
		if bindErr := baseViper.Unmarshal(&baseCfg); bindErr != nil {
			err = fmt.Errorf("基础配置绑定结构体失败：%w", bindErr)
			return
		}

		if optsCopy.Mode == "" {
			optsCopy.Mode = baseCfg.GetConfigMode()
		}

		// 根据模式创建对应泛型适配器实例
		switch optsCopy.Mode {
		case ModeLocal:
			adapter = &LocalConfig[T, U]{
				opts: optsCopy,
				v:    viper.New(),
				base: baseViper,
			}
		case ModeEtcd: // 只有 etcd 模式才会执行
			etcdCfgFromBase := baseCfg.GetEtcdConfig()
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

			adapter = &EtcdConfig[T, U]{
				opts: optsCopy,
				v:    viper.New(),
				base: baseViper,
			}
		default: // 默认走本地配置
			adapter = &LocalConfig[T, U]{
				opts: optsCopy,
				v:    viper.New(),
				base: baseViper,
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
	return adapter, nil
}

/**
 * @description:获取全局泛型适配器实例
 * @return {*}
 */
func GetGlobalAdapter[T IBaseConfig, U ICommonConfig]() (ConfigAdapter[T, U], error) {
	if globalAdapter == nil {
		return nil, fmt.Errorf("全局配置适配器未初始化，请先调用Init[T,U]()")
	}
	// 类型断言为对应泛型适配器
	adapter, ok := globalAdapter.(ConfigAdapter[T, U])
	if !ok {
		return nil, fmt.Errorf("全局适配器类型不匹配，预期[%T]，实际[%T]", adapter, globalAdapter)
	}
	return adapter, nil
}
