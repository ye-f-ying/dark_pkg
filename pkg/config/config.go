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
}

// ICommonConfig 公共配置泛型约束（空接口，同上）
type ICommonConfig interface {
	GetMysql() *MYSQL
	GetDebug() bool
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

type CommonConfig struct {
	Debug bool   `mapstructure:"debug"`
	MySql *MYSQL `mapstructure:"mysql"`
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
