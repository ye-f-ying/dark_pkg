/*
 * @Author: yeying
 * @Date: 2026-02-04 14:09:10
 * @FilePath: /dark_pkg/pkg/config/mysql_config.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package config

// MYSQL 配置
type MYSQL struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"db_name"`

	IsRead   bool   `mapstructure:"is_read"`
	ReadHost string `mapstructure:"read_host"`
	ReadPort int    `mapstructure:"read_port"`
	ReadUser string `mapstructure:"read_user"`

	ReadPassword string `mapstructure:"read_password"`
	ReadDBName   string `mapstructure:"read_db_name"`

	// 连接池配置
	MaxOpenConns    int `mapstructure:"max_open_conns"`
	MaxIdleConns    int `mapstructure:"max_idle_conns"`
	ConnMaxLifeTime int `mapstructure:"conn_max_life_time"`
	ConnMaxIdleTime int `mapstructure:"conn_max_idle_time"`

	// 公共配置
	Charset   string `mapstructure:"charset"`   //	字符集
	ParseTime bool   `mapstructure:"parseTime"` //	解析时间
	Loc       string `mapstructure:"loc"`       //	时区
}
