/*
 * @Author: yeying
 * @Date: 2026-02-05 11:43:21
 * @FilePath: /dark_pkg/pkg/zap/hlog_kitex_logger.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package zap

import (
	"context"
	"io"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/kitex/pkg/klog"
)

// HlogKitexLogger 绑定到 Kitex
type HlogKitexLogger struct{}

func (l *HlogKitexLogger) Debug(args ...interface{})  { hlog.Debug(args...) }
func (l *HlogKitexLogger) Info(args ...interface{})   { hlog.Info(args...) }
func (l *HlogKitexLogger) Warn(args ...interface{})   { hlog.Warn(args...) }
func (l *HlogKitexLogger) Error(args ...interface{})  { hlog.Error(args...) }
func (l *HlogKitexLogger) Fatal(args ...interface{})  { hlog.Fatal(args...) }
func (l *HlogKitexLogger) Notice(args ...interface{}) { hlog.Notice(args...) }
func (l *HlogKitexLogger) Trace(args ...interface{})  { hlog.Trace(args...) }

func (l *HlogKitexLogger) Debugf(format string, a ...interface{})  { hlog.Debugf(format, a...) }
func (l *HlogKitexLogger) Infof(format string, a ...interface{})   { hlog.Infof(format, a...) }
func (l *HlogKitexLogger) Warnf(format string, a ...interface{})   { hlog.Warnf(format, a...) }
func (l *HlogKitexLogger) Errorf(format string, a ...interface{})  { hlog.Errorf(format, a...) }
func (l *HlogKitexLogger) Fatalf(format string, a ...interface{})  { hlog.Fatalf(format, a...) }
func (l *HlogKitexLogger) Noticef(format string, a ...interface{}) { hlog.Noticef(format, a...) }
func (l *HlogKitexLogger) Tracef(format string, a ...interface{})  { hlog.Tracef(format, a...) }

func (l *HlogKitexLogger) CtxDebugf(ctx context.Context, f string, a ...interface{}) {
	hlog.CtxDebugf(ctx, f, a...)
}
func (l *HlogKitexLogger) CtxInfof(ctx context.Context, f string, a ...interface{}) {
	hlog.CtxInfof(ctx, f, a...)
}
func (l *HlogKitexLogger) CtxWarnf(ctx context.Context, f string, a ...interface{}) {
	hlog.CtxWarnf(ctx, f, a...)
}
func (l *HlogKitexLogger) CtxErrorf(ctx context.Context, f string, a ...interface{}) {
	hlog.CtxErrorf(ctx, f, a...)
}
func (l *HlogKitexLogger) CtxFatalf(ctx context.Context, f string, a ...interface{}) {
	hlog.CtxFatalf(ctx, f, a...)
}
func (l *HlogKitexLogger) CtxNoticef(ctx context.Context, f string, a ...interface{}) {
	hlog.CtxNoticef(ctx, f, a...)
}
func (l *HlogKitexLogger) CtxTracef(ctx context.Context, f string, a ...interface{}) {
	hlog.CtxTracef(ctx, f, a...)
}
func (l *HlogKitexLogger) SetLogger(v hlog.FullLogger) {
	hlog.SetLogger(v)
}

func (l *HlogKitexLogger) SetLevel(lv klog.Level) {
	hlog.SetLevel(hLevelToKlogLevel(lv))
}
func (l *HlogKitexLogger) SetOutput(w io.Writer) {
	hlog.SetOutput(w)
}

// BindKitexLogger 注入到 Kitex
func BindKitexLogger() {
	klog.SetLogger(&HlogKitexLogger{})
}

func hLevelToKlogLevel(level klog.Level) hlog.Level {
	var lvl hlog.Level
	switch level {
	case klog.LevelTrace:
		lvl = hlog.LevelTrace
	case klog.LevelDebug:
		lvl = hlog.LevelDebug
	case klog.LevelInfo:
		lvl = hlog.LevelInfo
	case klog.LevelWarn:
		lvl = hlog.LevelWarn
	case klog.LevelNotice:
		lvl = hlog.LevelNotice
	case klog.LevelError:
		lvl = hlog.LevelError
	case klog.LevelFatal:
		lvl = hlog.LevelFatal
	default:
		lvl = hlog.LevelWarn
	}
	return lvl
}
