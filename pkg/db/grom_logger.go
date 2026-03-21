/*
 * @Author: yeying
 * @Date: 2026-02-05 14:06:13
 * @FilePath: /dark_pkg/pkg/db/grom_logger.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package db

import (
	"context"
	"errors"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type GormLogger struct {
}

// LogMode 这里可以控制日志级别，这里固定为 Error
func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

/**
 * @description: info 日志
 * @param {context.Context} ctx
 * @param {string} s
 * @param {...interface{}} args
 * @return {*}
 */
func (l *GormLogger) Info(ctx context.Context, s string, args ...interface{}) {
	hlog.Infof(s, args...)
}

/**
 * @description:
 * @param {context.Context} ctx
 * @param {string} s
 * @param {...interface{}} args
 * @return {*}
 */
func (l *GormLogger) Warn(ctx context.Context, s string, args ...interface{}) {
	hlog.Warnf(s, args...)
}

/**
 * @description:Debug日志
 * @param {context.Context} ctx
 * @param {string} s
 * @param {...interface{}} args
 * @return {*}
 */
func (l *GormLogger) Debug(ctx context.Context, s string, args ...interface{}) {
	hlog.Debugf(s, args...)
}

/**
 * @description: Error 日志
 * @param {context.Context} ctx
 * @param {string} s
 * @param {...interface{}} args
 * @return {*}
 */
func (l *GormLogger) Error(ctx context.Context, s string, args ...interface{}) {
	hlog.Errorf(s, args...)
}

/**
 * @description: Trace 用于记录 SQL 执行
 * @return {*}
 */
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, rows := fc()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) { //错误输出屏蔽为查询到的错误
		hlog.Errorf("[err:%v] [sql:%s] [rows:%d] [elapsed:%d] gorm sql erro", err, sql, rows, time.Since(begin).Milliseconds())
	} else {
		hlog.Tracef("[sql:%s] [rows:%d] [elapsed:%d] ", sql, rows, time.Since(begin).Milliseconds())
	}
}
