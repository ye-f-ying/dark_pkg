/*
 * @Author: yeying
 * @Date: 2026-02-05 15:22:55
 * @FilePath: /dark_pkg/pkg/open_telemetry/otel_init.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package open_telemetry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/ye-f-ying/dark_pkg/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// 全局Tracer（Hertz/Kitex共用）
var globalTracer trace.Tracer

/**
 * @description: 初始化OTel
 * @param {context.Context} isWrite  上下文（用于OTLP Exporter初始化）
 * @param {*config.OpenTelemetryConfig} cfg OTel配置（仅需关注ServiceName/ExporterAddr/Enable等）
 * @return {*}优雅关闭函数 + 错误
 */
func InitOpenTelemetry(ctx context.Context, cfg *config.OpenTelemetryConfig) (func(), error) {
	if !cfg.Enable || cfg.ExporterAddr == "" {
		hlog.Info("OpenTelemetry is disabled, skip init")
		return func() {}, nil
	}

	// 核心配置校验 -- 服务id也做服务名称
	serverName := config.GetServerID()
	if serverName == "" {
		return nil, errors.New("otel service name is required")
	}
	// 采样率默认全采样（0-1，1为100%）
	if cfg.SamplerRatio <= 0 || cfg.SamplerRatio > 1 {
		cfg.SamplerRatio = 1.0
	}

	// 创建OTLP Exporter（唯一导出方式）
	exporter, err := newOTLPExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create otlp exporter failed: %w", err)
	}

	// 创建服务标识（Resource）：附加到所有Trace Span中
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serverName),                // 服务名（核心）
			semconv.DeploymentEnvironment(cfg.Environment), // 环境（dev/test/prod）
		),
		resource.WithSchemaURL(semconv.SchemaURL), // 标准Schema
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource failed: %w", err)
	}

	// 创建全局TracerProvider（Hertz/Kitex共用）
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second), // 批量导出超时
			sdktrace.WithMaxExportBatchSize(100),     // 批量导出最大数量
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplerRatio)), // 采样策略
	)

	// 设置全局OTel配置（链路透传核心）
	otel.SetTracerProvider(tp)
	// 文本传播器：兼容HTTP/RPC，保证Hertz->Kitex链路透传
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // 传递TraceID/SpanID
		propagation.Baggage{},      // 传递业务附加信息
	))

	// 初始化全局Tracer
	globalTracer = tp.Tracer(serverName)

	hlog.Infof("OpenTelemetry init success (OTLP only) | service: %s | otlp addr: %s | sampler: %.2f",
		serverName, cfg.ExporterAddr, cfg.SamplerRatio)

	// 优雅关闭函数（程序退出时调用）
	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			hlog.Errorf("OpenTelemetry shutdown failed: %v", err)
		} else {
			hlog.Info("OpenTelemetry shutdown success")
		}
	}

	return shutdown, nil
}

// 获取全局Tracer（业务中自定义Span时使用）
func GetGlobalTracer() trace.Tracer {
	return globalTracer
}

// 创建OTLP gRPC Exporter（仅支持OTLP，无Jaeger）
func newOTLPExporter(ctx context.Context, cfg *config.OpenTelemetryConfig) (sdktrace.SpanExporter, error) {
	// OTLP默认地址（gRPC端口4317，HTTP端口4318）
	if cfg.ExporterAddr == "" {
		cfg.ExporterAddr = "127.0.0.1:4317"
	}

	// OTLP Exporter配置项
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.ExporterAddr), // OTLP Collector地址
	}

	// 开发环境：跳过TLS验证（生产环境需启用）
	// 生产环境请移除WithInsecure，添加TLS配置（如下注释）
	opts = append(opts, otlptracegrpc.WithInsecure())

	// 生产环境TLS配置示例（替换WithInsecure）：
	// creds, err := credentials.NewClientTLSFromFile("/path/to/cert.pem", "otlp-collector")
	// if err != nil {
	// 	return nil, fmt.Errorf("load tls cert failed: %w", err)
	// }
	// opts = append(opts, otlptracegrpc.WithTLSCredentials(creds))

	// 创建OTLP Exporter
	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("init otlp exporter failed (addr: %s): %w", cfg.ExporterAddr, err)
	}

	return exporter, nil
}
