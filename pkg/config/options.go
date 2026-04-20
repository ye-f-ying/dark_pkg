/*
 * @Date: 2026-04-15 15:42:54
 * @LastEditTime: 2026-04-20 16:04:41
 * @FilePath: /dark_pkg/pkg/config/options.go
 * @Description:
 */
package config

import "sync"

// ConfigOptions 配置适配器选项（扩展IsWrite标记写入命令）
type ConfigOptions struct {
	Mode             ConfigMode // 运行模式：local/etcd
	ConfigPath       string     // 基础配置文件路径
	EtcdAddrs        []string   // etcd集群地址
	EtcdCommonKey    string     // etcd公共配置Key
	EtcdTimeout      int64      // etcd超时时间
	IsWrite          bool       // 是否是配置写入命令：true则将本地common.yml写入etcd
	EtcdUser         string
	EtcdPwd          string
	protectLocalKeys []string
}

// 定义全局泛型实例的持有者，避免包级泛型（Go不支持包级泛型变量）
var (
	globalAdapter any       // 泛型适配器实例（任意实现ConfigAdapter[T]的类型）
	globalConfig  any       // 泛型适配器实例（任意实现T的类型 通过接口获取公共设置）
	once          sync.Once // 单例初始化锁
	defaultOpts   = &ConfigOptions{
		Mode:             DefaultMode,
		ConfigPath:       DefaultConfigPath,
		EtcdCommonKey:    DefaultEtcdCommonKey,
		EtcdTimeout:      DefaultEtcdTimeout,
		IsWrite:          false,
		EtcdUser:         "",
		EtcdPwd:          "",
		protectLocalKeys: defaultProtectLocalKeys,
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
 * @description:本地配置文件路径
 * @param {string} path
 * @return {*}
 */
func WithConfigPath(path string) ConfigOption {
	return func(o *ConfigOptions) { o.ConfigPath = path }
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
 * @description:设置不被覆盖的私有数据 -- 注意会覆盖默认的私有配置 只有etcd 模式下有效
 * @param {[]string} plk
 * @return {*}
 */
func WithProtectLocalKeys(plk []string) ConfigOption {
	return func(o *ConfigOptions) { o.protectLocalKeys = plk }
}
