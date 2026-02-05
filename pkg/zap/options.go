/*
 * @Author: yeying
 * @Date: 2026-02-05 11:34:09
 * @FilePath: /dark_pkg/pkg/zap/options.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package zap

import (
	"os"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Option interface {
	apply(cfg *config)
}

type ExtraKey string

type option func(cfg *config)

func (fn option) apply(cfg *config) {
	fn(cfg)
}

type CoreConfig struct {
	Enc zapcore.Encoder
	Ws  zapcore.WriteSyncer
	Lvl zapcore.LevelEnabler
}

type config struct {
	extraKeys     []ExtraKey
	coreConfigs   []CoreConfig
	zapOpts       []zap.Option
	extraKeyAsStr bool
}

/**
 * @description: defaultCoreConfig 默认 zapcore 配置：json 编码器、原子级别、stdout 写入同步器
 * @return {*}
 */
func defaultCoreConfig() *CoreConfig {
	// default log encoder
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	// default log level
	lvl := zap.NewAtomicLevelAt(zap.InfoLevel)
	// default write syncer stdout
	ws := zapcore.AddSync(os.Stdout)

	return &CoreConfig{
		Enc: enc,
		Ws:  ws,
		Lvl: lvl,
	}
}

/**
 * @description:默认配置
 * @return {*}
 */
func defaultConfig() *config {
	return &config{
		coreConfigs:   []CoreConfig{*defaultCoreConfig()},
		zapOpts:       []zap.Option{},
		extraKeyAsStr: false,
	}
}

// WithCoreEnc zapcore encoder
func WithCoreEnc(enc zapcore.Encoder) Option {
	return option(func(cfg *config) {
		cfg.coreConfigs[0].Enc = enc
	})
}

// WithCoreWs zapcore write syncer
func WithCoreWs(ws zapcore.WriteSyncer) Option {
	return option(func(cfg *config) {
		cfg.coreConfigs[0].Ws = ws
	})
}

// WithCoreLevel zapcore log level
func WithCoreLevel(lvl zap.AtomicLevel) Option {
	return option(func(cfg *config) {
		cfg.coreConfigs[0].Lvl = lvl
	})
}

// WithCores zapcore
func WithCores(coreConfigs ...CoreConfig) Option {
	return option(func(cfg *config) {
		cfg.coreConfigs = coreConfigs
	})
}

// WithZapOptions add origin zap option
func WithZapOptions(opts ...zap.Option) Option {
	return option(func(cfg *config) {
		cfg.zapOpts = append(cfg.zapOpts, opts...)
	})
}

// WithExtraKeys allow you log extra values from context
func WithExtraKeys(keys []ExtraKey) Option {
	return option(func(cfg *config) {
		for _, k := range keys {
			if !InArray(k, cfg.extraKeys) {
				cfg.extraKeys = append(cfg.extraKeys, k)
			}
		}
	})
}

// WithExtraKeyAsStr convert extraKey to a string type when retrieving value from context
// Not recommended for use, only for compatibility with certain situations
//
// For more information, refer to the documentation at
// `https://pkg.go.dev/context#WithValue`
func WithExtraKeyAsStr() Option {
	return option(func(cfg *config) {
		cfg.extraKeyAsStr = true
	})
}

// InArray check if a string in a slice
func InArray(key ExtraKey, arr []ExtraKey) bool {
	for _, k := range arr {
		if k == key {
			return true
		}
	}
	return false
}

func hLevelToZapLevel(level hlog.Level) zapcore.Level {
	var lvl zapcore.Level
	switch level {
	case hlog.LevelTrace, hlog.LevelDebug:
		lvl = zap.DebugLevel
	case hlog.LevelInfo:
		lvl = zap.InfoLevel
	case hlog.LevelWarn, hlog.LevelNotice:
		lvl = zap.WarnLevel
	case hlog.LevelError:
		lvl = zap.ErrorLevel
	case hlog.LevelFatal:
		lvl = zap.FatalLevel
	default:
		lvl = zap.WarnLevel
	}
	return lvl
}
