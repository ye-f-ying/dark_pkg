/*
 * @Author: yeying
 * @Date: 2026-02-04 14:06:30
 * @FilePath: /dark_pkg/pkg/config/config.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package config

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
	ServerID   string      `mapstructure:"server_id"`   // 服务ID/名称
	MachineID  uint16      `mapstructure:"machine_id"`  // 分布式ID
	EtcdConfig *ETCDConfig `mapstructure:"etcd_config"` // etcd 配置
	ConfMode   ConfigMode  `mapstructure:"mode_config"` // 模式配置
	LogDir     string      `mapstructure:"log_dir"`     // 日志目录
}

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
