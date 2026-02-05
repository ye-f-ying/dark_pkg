/*
 * @Author: yeying
 * @Date: 2026-02-05 14:08:09
 * @FilePath: /dark_pkg/pkg/db/mysql_config.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package db

import (
	"sync"

	"github.com/ye-f-ying/dark_pkg/pkg/config"
)

var (
	msqlConf     *MYSQLConfig
	msqlConfOnce sync.Once
)

func GetMYSQLConfig() *MYSQLConfig {
	msqlConfOnce.Do(func() {
		msqlConf = &MYSQLConfig{}
		gaCfg, _ := config.GetGlobalAdapter[config.IBaseConfig, config.ICommonConfig]()
		if gaCfg == nil {
			panic("mysql not configured")
		}
		_, mysqlCfg, _ := gaCfg.GetConfig()
		if mysqlCfg == nil || mysqlCfg.GetMysql() == nil {
			panic("mysql not configured")
		}
		cfg := mysqlCfg.GetMysql()
		msqlConf.IsSlave = cfg.IsRead
		msqlConf.Master.Host = cfg.Host
		msqlConf.Master.Prot = cfg.Prot
		msqlConf.Master.User = cfg.User
		msqlConf.Master.Password = cfg.Password
		msqlConf.Master.DBName = cfg.DBName
		msqlConf.Master.MaxOpenConns = cfg.MaxOpenConns
		msqlConf.Master.MaxIdleConns = cfg.MaxIdleConns
		msqlConf.Master.ConnMaxLifeTime = cfg.ConnMaxLifeTime
		msqlConf.Master.ConnMaxIdleTime = cfg.ConnMaxIdleTime
		msqlConf.Master.Charset = cfg.Charset
		msqlConf.Master.ParseTime = cfg.ParseTime
		msqlConf.Master.Loc = cfg.Loc

		if cfg.IsRead {
			var slave SQLConfig
			slave.Host = cfg.ReadHost
			slave.Prot = cfg.ReadProt
			slave.User = cfg.ReadUser
			slave.Password = cfg.ReadPassword
			slave.DBName = cfg.ReadDBName
			slave.MaxOpenConns = cfg.MaxOpenConns
			slave.MaxIdleConns = cfg.MaxIdleConns
			slave.ConnMaxLifeTime = cfg.ConnMaxLifeTime
			slave.ConnMaxIdleTime = cfg.ConnMaxIdleTime
			slave.Charset = cfg.Charset
			slave.ParseTime = cfg.ParseTime
			slave.Loc = cfg.Loc
			msqlConf.Slaves = append(msqlConf.Slaves, slave)
		}

	})
	return msqlConf
}

type MYSQLConfig struct {
	Master  SQLConfig   //主库配置
	Slaves  []SQLConfig //从库配置
	IsSlave bool        //是否开启主从
}

// 数据库配置
type SQLConfig struct {
	Host     string `mapstructure:"host"`     //IP地址
	Prot     int    `mapstructure:"prot"`     //端口
	User     string `mapstructure:"user"`     //用户名
	Password string `mapstructure:"password"` //密码
	DBName   string `mapstructure:"db_name"`  //数据库名称

	MaxOpenConns    int `mapstructure:"maxOpenConns"`    //最大连接数
	MaxIdleConns    int `mapstructure:"maxIdleConns"`    //最大空闲连接数
	ConnMaxLifeTime int `mapstructure:"connMaxLifeTime"` //连接最大生命周期
	ConnMaxIdleTime int `mapstructure:"connMaxIdleTime"` //连接最大空闲时间

	Charset   string `mapstructure:"charset"`   //	字符集
	ParseTime bool   `mapstructure:"parseTime"` //	解析时间
	Loc       string `mapstructure:"loc"`       //	时区

}
