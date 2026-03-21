/*
 * @Author: yeying
 * @Date: 2026-02-05 14:07:09
 * @FilePath: /dark_pkg/pkg/db/grom.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package db

import (
	"fmt"

	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

var gromDBMYSQL *gorm.DB
var gromMYSQLDBOnce sync.Once

/**
 * @description: 初始化GromPGSQL
 * @param {MYSQLConfi} cfg
 * @return {*}
 */
func InitGromMYSQL() error {
	config := GetMYSQLConfig()
	if config == nil {
		return fmt.Errorf("数据库配置不能为空！请先配置数据库配置")
	}

	var errInfo error
	gromMYSQLDBOnce.Do(func() {
		MaxOpenConns := config.Master.MaxOpenConns
		MaxIdleConns := config.Master.MaxIdleConns
		ConnMaxLifeTime := config.Master.ConnMaxLifeTime
		ConnMaxIdleTime := config.Master.ConnMaxIdleTime
		if MaxOpenConns <= 0 {
			MaxOpenConns = 100
		}
		if MaxIdleConns <= 0 {
			MaxOpenConns = 20
		}
		if ConnMaxLifeTime <= 0 {
			MaxOpenConns = 1800
		}
		if ConnMaxIdleTime <= 0 {
			MaxOpenConns = 300
		}
		masterDNS := generateDNSMYSQL(config.Master)
		master, err := gorm.Open(mysql.New(mysql.Config{
			DSN: masterDNS,
		}), &gorm.Config{
			Logger: &GormLogger{},
		})
		if err != nil {
			errInfo = err
			return
		}
		//mysql 主从配置
		if config.IsSlave && len(config.Slaves) > 0 {
			slaves := make([]gorm.Dialector, len(config.Slaves))
			for index, slave := range config.Slaves {
				slaves[index] = mysql.New(mysql.Config{
					DSN: generateDNSMYSQL(slave),
				})
			}
			master.Use(dbresolver.Register(
				dbresolver.Config{
					Sources: []gorm.Dialector{mysql.New(mysql.Config{
						DSN: masterDNS,
					})},
					Replicas:          slaves,
					Policy:            dbresolver.RandomPolicy{},
					TraceResolverMode: true,
				},
			).SetMaxOpenConns(MaxOpenConns).
				SetMaxIdleConns(MaxIdleConns).
				SetConnMaxLifetime(time.Duration(ConnMaxLifeTime) * time.Second).
				SetConnMaxIdleTime(time.Duration(ConnMaxIdleTime) * time.Second),
			)
			masterDB, err := master.DB()
			if err != nil {
				errInfo = err
				return
			}
			err = masterDB.Ping()
			if err != nil {
				errInfo = err
				return
			}
			gromDBMYSQL = master
			return
		}

		masterDB, err := master.DB()
		if err != nil {
			errInfo = err
			return
		}
		//链接池配置
		masterDB.SetMaxOpenConns(MaxOpenConns)
		masterDB.SetMaxIdleConns(MaxIdleConns)
		masterDB.SetConnMaxLifetime(time.Duration(ConnMaxLifeTime) * time.Second)
		masterDB.SetConnMaxIdleTime(time.Duration(ConnMaxIdleTime) * time.Second)
		err = masterDB.Ping()
		if err != nil {
			errInfo = err
			return
		}
		gromDBMYSQL = master
	})
	return errInfo
}

/**
 * @description: 获取Grom 数据库对象
 * @return {*}
 */
func GetGromDBMYSQL() *gorm.DB {
	if gromDBMYSQL == nil {
		InitGromMYSQL()
	}
	return gromDBMYSQL
}

func generateDNSMYSQL(conf SQLConfig) string {
	if conf.Charset == "" {
		conf.Charset = "utf8mb4"
	}
	//目前强制用true 不然时间解析不了
	parseTime := "true"
	/*if !conf.ParseTime {
		parseTime = "false"
	}*/
	if conf.Loc == "" {
		conf.Loc = "Local"
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%s&loc=%s",
		conf.User,
		conf.Password,
		conf.Host,
		conf.Port,
		conf.DBName,
		conf.Charset,
		parseTime,
		conf.Loc)
}
