/*
 * @Author: yeying
 * @Date: 2026-02-05 11:29:13
 * @FilePath: /dark_pkg/pkg/zap/zap.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package zap

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/natefinch/lumberjack"
	pkgConfig "github.com/ye-f-ying/dark_pkg/pkg/config"
	"github.com/ye-f-ying/dark_pkg/pkg/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger     *Logger
	loggerOnce sync.Once
	logDir     string
	level      zapcore.Level
	logDirOnce sync.Once
)

const (
	MaxFileSize = 100  // 单个日志文件最大大小（MB），按大小切割阈值
	MaxBackups  = 100  // 日志文件最大备份数
	MaxAge      = 30   // 日志文件最大保留天数
	LogFilePerm = 0755 // 日志目录权限
	cutHour     = 0    // 每日切割时间（凌晨0点）
)

func GetLogger() *Logger {
	return logger
}

/**
 * @description: 初始化日志
 * @param {pkgConfig.IBaseConfig} baseCfg
 * @param {pkgConfig.ICommonConfig} cfg
 * @return {*}
 */
func InitZap(baseCfg pkgConfig.IBaseConfig, cfg pkgConfig.ICommonConfig) {
	loggerOnce.Do(func() {
		level = zap.DebugLevel
		if cfg != nil {
			level = zap.InfoLevel
		}
		if baseCfg != nil {
			logDir = baseCfg.GetLogDir()
		}

		dynamicLevel := zap.NewAtomicLevel()
		dynamicLevel.SetLevel(level)
		setLogger(dynamicLevel)
		go func() {
			for {
				// 计算当前时间到次日凌晨cutHour点的时间差
				now := time.Now()
				nextCut := time.Date(now.Year(), now.Month(), now.Day()+1, cutHour, 0, 0, 0, now.Location())
				waitDur := nextCut.Sub(now)
				// 等待到切割时间
				time.Sleep(waitDur)
				// 到达时间，主动切割所有日志文件
				setLogger(dynamicLevel)
			}
		}()

	})
}

func setLogger(dynamicLevel zap.AtomicLevel) {
	logger = NewLogger(
		WithCores([]CoreConfig{
			{
				Enc: zapcore.NewJSONEncoder(humanEncoderConfig()),
				Ws:  zapcore.AddSync(os.Stdout),
				Lvl: dynamicLevel,
			},
			{
				Enc: zapcore.NewJSONEncoder(humanEncoderConfig()),
				Ws:  getDailyWriteSyncer("all"),
				Lvl: zap.NewAtomicLevelAt(zapcore.DebugLevel),
			},
			{
				Enc: zapcore.NewJSONEncoder(humanEncoderConfig()),
				Ws:  getDailyWriteSyncer("debug"),
				Lvl: zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
					return lev == zap.DebugLevel
				}),
			},
			{
				Enc: zapcore.NewJSONEncoder(humanEncoderConfig()),
				Ws:  getDailyWriteSyncer("info"),
				Lvl: zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
					return lev == zap.InfoLevel
				}),
			},
			{
				Enc: zapcore.NewJSONEncoder(humanEncoderConfig()),
				Ws:  getDailyWriteSyncer("warn"),
				Lvl: zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
					return lev == zap.WarnLevel
				}),
			},
			{
				Enc: zapcore.NewJSONEncoder(humanEncoderConfig()),
				Ws:  getDailyWriteSyncer("error"),
				Lvl: zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
					return lev >= zap.ErrorLevel
				}),
			},
		}...),
	)
	hlog.SetLogger(logger)
	klog.SetLogger(&HlogKitexLogger{}) // 合并 klog 到hlog
}

func humanEncoderConfig() zapcore.EncoderConfig {
	cfg := testEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.EncodeDuration = zapcore.StringDurationEncoder
	return cfg
}

func getDailyWriteSyncer(prefix string) zapcore.WriteSyncer {
	today := time.Now().Format("2006-01-02")
	dir := strings.TrimSuffix(logDir, "/")
	if dir == "" {
		dir = "./logs"
	}

	err := utils.MkdirIfNotExist(dir)
	if err != nil {
		hlog.Errorf("创建文件夹[%s]失败！%w", dir, err)
	}

	file := fmt.Sprintf("%s/%s-%s.log", dir, prefix, today) // ← 文件名格式

	lj := &lumberjack.Logger{
		Filename:   file,
		MaxSize:    MaxFileSize,
		MaxBackups: MaxBackups, // 最大备份数
		MaxAge:     MaxAge,     // 最多保留30天
		Compress:   true,       // 压缩旧日志
		LocalTime:  true,
	}

	return zapcore.AddSync(lj)
}

func testEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "name",
		TimeKey:        "ts",
		CallerKey:      "caller",
		FunctionKey:    "func",
		StacktraceKey:  "stacktrace",
		LineEnding:     "\n",
		EncodeTime:     zapcore.EpochTimeEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}
