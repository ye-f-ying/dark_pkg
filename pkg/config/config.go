/*
 * @Author: yeying
 * @Date: 2026-02-04 14:06:30
 * @FilePath: /dark_pkg/pkg/config/config.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package config

import (
	"flag"

	"github.com/spf13/viper"
)

// IBaseConfig 基础配置泛型约束
type IBaseConfig interface {
	GetServerID() string        //服务器Id/名称
	GetMachineID() uint16       // 分布式ID 或者通过etcd 等自动分配
	GetEtcdConfig() *ETCDConfig // etcd 配置中心
	GetConfigMode() ConfigMode  // 配置加载模式
	GetLogDir() string
}

// ICommonConfig 公共配置泛型约束（空接口，同上）
type ICommonConfig interface {
	GetMysql() *MYSQL
	GetDebug() bool
	GetOpenTelemetry() *OpenTelemetryConfig
}

type BaseConfig struct {
	ServerID   string      `mapstructure:"server_id" default:"server_name"` // 服务ID/名称
	MachineID  uint16      `mapstructure:"machine_id"  default:"1"`         // 分布式ID
	EtcdConfig *ETCDConfig `mapstructure:"etcd_config"`                     // etcd 配置
	ConfMode   ConfigMode  `mapstructure:"mode_config" default:"local"`     // 模式配置
	LogDir     string      `mapstructure:"log_dir" default:"./logs"`        // 日志目录
}

// 命令行参数
var (
	cliBaseConfigPath string // --config 配置文件路径
	cliConfigPath     string // --common-config 配置文件路径
	cliConfigWrite    int    // --config-Write 配置文件路径
	// 命令行参数（和 BaseConfig 字段对应）
	cliServerID   string // --server-id server_id
	cliMachineID  int    // --machine-id machine_id
	cliLogDir     string // --log-dir log_dir
	cliModeConfig string // --mode-config server_id

	// etcd
	cliETCDAddress              string   //  --etcd-address
	cliETCDEndpoints            []string // --etcd-endpoints
	cliETCDUsername             string   //  --etcd-username
	cliETCDPassword             string   //  --etcd-password
	cliETCDDialTimeout          int64    // --etcd-dial-timeout
	cliETCDDialKeepAliveTime    int      // --etcd-dial-keep-alive-time
	cliETCDDialKeepAliveTimeout int      // --etcd-dial-keep-alive-timeout
	cliETCDCommonKey            string   // --etcd-common-key
)

/**
 * @description: 默认服务ID
 * @return {*}
 */
func (m *BaseConfig) GetServerID() string {
	return m.ServerID
}

/**
 * @description: 默认分布式ID
 * @return {*}
 */
func (m *BaseConfig) GetMachineID() uint16 {
	return m.MachineID
}

/**
 * @description: etcd配置中心连接信息
 * @return {*}
 */
func (m *BaseConfig) GetEtcdConfig() *ETCDConfig {
	return m.EtcdConfig
}

/**
 * @description: etcd配置中心连接信息
 * @return {*}
 */
func (m *BaseConfig) GetConfigMode() ConfigMode {
	return m.ConfMode
}

/**
 * @description: 获取日志路径
 * @return {*}
 */
func (m *BaseConfig) GetLogDir() string {
	return m.LogDir
}

type CommonConfig struct {
	Debug            bool                 `mapstructure:"debug"`
	MySql            *MYSQL               `mapstructure:"mysql"`
	OpenTelemetryCfg *OpenTelemetryConfig `mapstructure:"open_telemetry"`
}

/**
 * @description: 默认数据库配置
 * @return {*}
 */
func (m *CommonConfig) GetMysql() *MYSQL {
	return m.MySql
}

/**
 * @description: 是否启用debug模式
 * @return {*}
 */
func (m *CommonConfig) GetDebug() bool {
	return m.Debug
}

/**
 * @description:OpenTelemetry 设置
 * @return {*}
 */
func (m *CommonConfig) GetOpenTelemetry() *OpenTelemetryConfig {
	return m.OpenTelemetryCfg
}

/**
 * @description: 基础配置
 * @return {*}
 */
func GetBaseConfig() IBaseConfig {
	cfg, _ := GetGlobalAdapter[IBaseConfig, ICommonConfig]()
	if cfg == nil {
		return nil
	}
	baseCfg, _, _ := cfg.GetConfig()
	return baseCfg
}

/**
 * @description: 公共配置
 * @return {*}
 */
func GetCommonConfig() ICommonConfig {
	cfg, _ := GetGlobalAdapter[IBaseConfig, ICommonConfig]()
	if cfg == nil {
		return nil
	}
	_, commonCfg, _ := cfg.GetConfig()
	return commonCfg
}

/**
 * @description: 服务器ID/名称
 * @return {*}
 */
func GetServerID() string {
	baseCfg := GetBaseConfig()
	if baseCfg == nil {
		return ""
	}
	return baseCfg.GetServerID()
}

/**
 * @description: 服务分布式ID
 * @return {*}
 */
func GetMachineID() uint16 {
	baseCfg := GetBaseConfig()
	if baseCfg == nil {
		return 0
	}
	return baseCfg.GetMachineID()
}

func initFlag() {
	// 基础配置参数
	flag.StringVar(&cliBaseConfigPath, "config", "", "配置文件路径（默认 ）")
	flag.StringVar(&cliConfigPath, "common-config", "", "公共配置文件路径（默认 ）")
	flag.IntVar(&cliConfigWrite, "config-write", -1, "是否将配置文件写入etcd")
	flag.StringVar(&cliServerID, "server-id", "", "服务器ID（覆盖配置文件 server_id）")
	flag.IntVar(&cliMachineID, "machine-id", 0, "分布式ID机器标识（覆盖配置文件 machine_id）")
	flag.StringVar(&cliLogDir, "log-dir", "", "日志目录（覆盖配置文件 log_dir）")
	flag.StringVar(&cliModeConfig, "mode-config", "", "日志目录（覆盖配置文件 mode_config）")

	// etcd
	flag.StringVar(&cliETCDAddress, "etcd-address", "", "访问地址+注册端口（覆盖配置文件 etcd_config.address）")
	flag.Func("etcd-endpoints", "ETCD集群地址（可多次指定，覆盖 etcd_config.endpoints）", func(s string) error {
		cliETCDEndpoints = append(cliETCDEndpoints, s)
		return nil
	})
	flag.StringVar(&cliETCDUsername, "etcd-username", "", "etcd 用户名（覆盖配置文件 etcd_config.username")
	flag.StringVar(&cliETCDPassword, "etcd-password", "", "etcd 密码（覆盖配置文件 etcd_config.password")
	flag.Int64Var(&cliETCDDialTimeout, "etcd-dial-timeout", -999999, "连接超时时间（秒）（覆盖配置文件 etcd_config.dial_timeout")
	flag.IntVar(&cliETCDDialKeepAliveTime, "etcd-dial-keep-alive-time", -999999, "客户端发起 KeepAlive PING 的周期（秒）（覆盖配置文件 etcd_config.dial_keep_alive_time）")
	flag.IntVar(&cliETCDDialKeepAliveTimeout, "etcd-dial-keep-alive-timeout", -999999, "客户端发出 KeepAlive 探测后，等待服务端响应的超时时间（覆盖配置文件 etcd_config.dial_keep_alive_timeout）")
	flag.StringVar(&cliETCDCommonKey, "etcd-common-key", "", "etcd公共配置Key（覆盖配置文件 etcd_config.common_key）")
	// 解析命令行参数
	flag.Parse()

}

func localFlag(baseViper *viper.Viper) {
	if baseViper == nil {
		return
	}
	if cliServerID != "" {
		baseViper.Set("server_id", cliServerID)
	}

	if cliMachineID <= 0 {
		baseViper.Set("machine_id", cliMachineID)
	}

	if cliModeConfig != "" {
		baseViper.Set("mode_config", cliModeConfig)
	}

	if cliLogDir != "" {
		baseViper.Set("log_dir", cliLogDir)
	}

	// etcd_config
	if cliETCDAddress != "" {
		baseViper.Set("etcd_config.address", cliETCDAddress)
	}
	if len(cliETCDEndpoints) > 0 {
		baseViper.Set("etcd_config.endpoints", cliETCDEndpoints)
	}
	if cliETCDUsername != "" {
		baseViper.Set("etcd_config.username", cliETCDUsername)
	}
	if cliETCDPassword != "" {
		baseViper.Set("etcd_config.password", cliETCDPassword)
	}
	if cliETCDDialTimeout != -999999 {
		baseViper.Set("etcd_config.dial_timeout", cliETCDDialTimeout)
	}
	if cliETCDDialKeepAliveTime != -999999 {
		baseViper.Set("etcd_config.dial_keep_alive_time", cliETCDDialKeepAliveTime)
	}
	if cliETCDDialKeepAliveTimeout != -999999 {
		baseViper.Set("etcd_config.dial_keep_alive_timeout", cliETCDDialKeepAliveTimeout)
	}
	if cliETCDCommonKey != "" {
		baseViper.Set("etcd_config.common_key", cliETCDCommonKey)
	}

}

func defaultConfig(baseViper *viper.Viper) {
	if baseViper == nil {
		return
	}
	baseViper.Set("server_id", "server")
	baseViper.Set("machine_id", 1)
	baseViper.Set("mode_config", ModeLocal)
	baseViper.Set("log_dir", "./logs")
}
