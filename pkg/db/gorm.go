/*
 * @Author: yeying
 * @Date: 2026-02-05 14:07:09
 * @FilePath: /dark_pkg/pkg/db/gorm.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package db

import (
	"fmt"
	"os"

	"sync"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

var gormDBMYSQL *gorm.DB
var gormMYSQLDBOnce sync.Once
var gormGetOnce sync.Once

/**
 * @description: 初始化GromPGSQL
 * @param {MYSQLConfi} cfg
 * @return {*}
 */
func InitGormMYSQL() error {
	config := GetMYSQLConfig()
	if config == nil {
		return fmt.Errorf("数据库配置不能为空！请先配置数据库配置")
	}

	var errInfo error
	gormMYSQLDBOnce.Do(func() {
		MaxOpenConns := config.Master.MaxOpenConns
		MaxIdleConns := config.Master.MaxIdleConns
		ConnMaxLifeTime := config.Master.ConnMaxLifeTime
		ConnMaxIdleTime := config.Master.ConnMaxIdleTime
		if MaxOpenConns <= 0 {
			MaxOpenConns = 100
		}
		if MaxIdleConns <= 0 {
			MaxIdleConns = 20
		}
		if ConnMaxLifeTime <= 0 {
			ConnMaxLifeTime = 1800
		}
		if ConnMaxIdleTime <= 0 {
			ConnMaxIdleTime = 300
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
			err = master.Use(dbresolver.Register(
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
			if err != nil {
				errInfo = err
				return
			}
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
			gormDBMYSQL = master
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
		gormDBMYSQL = master
	})
	return errInfo
}

/**
 * @description: 获取Grom 数据库对象
 * @return {*}
 */
func GetGormDBMYSQL() *gorm.DB {
	gormGetOnce.Do(func() {
		if err := InitGormMYSQL(); err != nil {
			fmt.Printf("init grom db error:%v ", err)
			os.Exit(1)
		}
	})
	return gormDBMYSQL
}

func generateDNSMYSQL(conf SQLConfig) string {
	// 字符集默认 utf8mb4
	if conf.Charset == "" {
		conf.Charset = "utf8mb4"
	}
	// 时区默认 Local
	if conf.Loc == "" {
		conf.Loc = "Local"
	}

	// 目前强制 true，不然时间解析不了
	conf.ParseTime = true
	loc, err := time.LoadLocation(conf.Loc) // Loc 是 string，driver.Config 需要 *time.Location
	if err != nil {
		loc = time.Local
	}
	mc := driver.Config{
		User:   conf.User,
		Passwd: conf.Password,
		Net:    "tcp",
		Addr:   fmt.Sprintf("%s:%d", conf.Host, conf.Port),
		DBName: conf.DBName,
		Params: map[string]string{"charset": conf.Charset}, // charset 通过 Params 传入
		// 其余 charset/timeout 参数按现有 DSN 字符串补齐
		ParseTime: conf.ParseTime,
		Loc:       loc,
	}
	return mc.FormatDSN()

	/*if conf.Charset == "" {
		conf.Charset = "utf8mb4"
	}
	// 目前强制用true 不然时间解析不了
	parseTime := "true"
	if !conf.ParseTime {
		parseTime = "false"
	}
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
		conf.Loc)*/

}
