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
	Prot     int    `mapstructure:"prot"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"db_name"`

	IsRead   bool   `mapstructure:"is_read"`
	ReadHost string `mapstructure:"read_host"`
	ReadProt int    `mapstructure:"read_prot"`
	ReadUser string `mapstructure:"read_user"`

	ReadPassword string `mapstructure:"read_password"`
	ReadDBName   string `mapstructure:"read_db_name"`

	// 连接池配置
	MaxOpenConns    int `mapstructure:"maxOpenConns"`
	MaxIdleConns    int `mapstructure:"maxIdleConns"`
	ConnMaxLifeTime int `mapstructure:"connMaxLifeTime"`
	ConnMaxIdleTime int `mapstructure:"connMaxIdleTime"`
}
