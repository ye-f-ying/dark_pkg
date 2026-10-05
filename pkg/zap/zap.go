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
	"sync/atomic"
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
	logger     atomic.Pointer[Logger] //*Logger
	loggerOnce sync.Once
	logDir     string
	level      zapcore.Level
)

const (
	MaxFileSize = 100  // 单个日志文件最大大小（MB），按大小切割阈值
	MaxBackups  = 100  // 日志文件最大备份数
	MaxAge      = 30   // 日志文件最大保留天数
	LogFilePerm = 0755 // 日志目录权限
	cutHour     = 0    // 每日切割时间（凌晨0点）
)

func GetLogger() *Logger {
	return logger.Load()
}

/**
 * @description:初始化日志
 * @param {pkgConfig.IConfig} cfg
 * @return {*}
 */
func InitZap(cfg pkgConfig.IConfig) {
	loggerOnce.Do(func() {
		level = zap.DebugLevel
		if cfg != nil && !cfg.GetDebug() {
			level = zap.InfoLevel
		}
		if cfg != nil {
			logDir = cfg.GetLogDir()
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

type rotation struct {
	lg    *Logger
	files []*lumberjack.Logger
}

var cur atomic.Value

func setLogger(dynamicLevel zap.AtomicLevel) {
	ljs := make([]*lumberjack.Logger, 0, 5)
	dailyWS := func(prefix string) zapcore.WriteSyncer {
		lj := newLumberjack(prefix) // 原来的 getDailyWriteSyncer 拆出，返回 *lumberjack.Logger
		ljs = append(ljs, lj)
		return zapcore.AddSync(lj)
	}
	//logger
	newLg := NewLogger(
		WithCores([]CoreConfig{
			{
				Enc: zapcore.NewJSONEncoder(humanEncoderConfig()),
				Ws:  zapcore.AddSync(os.Stdout),
				Lvl: dynamicLevel,
			},
			{
				Enc: zapcore.NewJSONEncoder(humanEncoderConfig()),
				Ws:  dailyWS("all"),
				Lvl: zap.NewAtomicLevelAt(zapcore.DebugLevel),
			},
			{
				Enc: zapcore.NewJSONEncoder(humanEncoderConfig()),
				Ws:  dailyWS("debug"),
				Lvl: zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
					return lev == zap.DebugLevel
				}),
			},
			{
				Enc: zapcore.NewJSONEncoder(humanEncoderConfig()),
				Ws:  dailyWS("info"),
				Lvl: zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
					return lev == zap.InfoLevel
				}),
			},
			{
				Enc: zapcore.NewJSONEncoder(humanEncoderConfig()),
				Ws:  dailyWS("warn"),
				Lvl: zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
					return lev == zap.WarnLevel
				}),
			},
			{
				Enc: zapcore.NewJSONEncoder(humanEncoderConfig()),
				Ws:  dailyWS("error"),
				Lvl: zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
					return lev >= zap.ErrorLevel
				}),
			},
		}...),
	)
	//hlog.SetLogger(logger)
	logger.Store(newLg)
	hlog.SetLogger(GetLogger())
	klog.SetLogger(&HlogKitexLogger{}) // 合并 klog 到hlog

	old := cur.Swap(&rotation{lg: newLg, files: ljs})
	if old != nil {
		go func(o *rotation) {
			o.lg.Sync() // ① flush 缓冲区（stdout 报 ENOTTY 属正常，忽略）
			for _, lj := range o.files {
				lj.Close() // ② 关闭旧文件句柄 ← 就是这个方法
			}
		}(old.(*rotation))
	}
}

func humanEncoderConfig() zapcore.EncoderConfig {
	cfg := testEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.EncodeDuration = zapcore.StringDurationEncoder
	return cfg
}

func newLumberjack(prefix string) *lumberjack.Logger {
	today := time.Now().Format("2006-01-02")
	dir := strings.TrimSuffix(logDir, "/")
	if dir == "" {
		dir = "./logs"
	}

	err := utils.MkdirIfNotExist(dir)
	if err != nil {
		hlog.Errorf("创建文件夹[%s]失败！%v", dir, err)
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
	return lj
}

/*
func getDailyWriteSyncer(prefix string) zapcore.WriteSyncer {
	return zapcore.AddSync(newLumberjack(prefix))
}*/

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
