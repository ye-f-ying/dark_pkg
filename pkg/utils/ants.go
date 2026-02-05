/*
 * @Author: yeying
 * @Date: 2026-02-05 14:28:59
 * @FilePath: /dark_pkg/pkg/utils/ants.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package utils

import (
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/panjf2000/ants/v2"
)

var (
	antsPool   *ants.Pool            // 重命名，语义更清晰
	antsOnce   sync.Once             // 保持单例初始化
	antsConfig = defaultAntsConfig() // 协程池默认配置
)

type AntsConfig struct {
	PoolSize int // 协程池大小，默认使用ants.DefaultAntsPoolSize
}

/**
 * @description: 设置协程池默认配置
 * @return {*}
 */
func defaultAntsConfig() AntsConfig {
	return AntsConfig{
		PoolSize: ants.DefaultAntsPoolSize,
	}
}

/**
 * @description: 自定义协程池配置（需在首次调用Go()前执行，建议程序启动时初始化）
 * @param {AntsConfig} cfg
 * @return {*}
 */
func SetAntsConfig(cfg AntsConfig) {
	if cfg.PoolSize <= 0 { // 防护：避免传入非法大小
		return
	}
	antsConfig = cfg
}

/**
 * @description:将任务提交到协程池，失败则降级为原生go执行，保证任务不丢失
 * @param {fun} ex 待执行的无参无返回函数
 * @return {*} 仅返回协程池初始化/配置的致命错误，任务提交失败会降级执行，不返回错误
 */
func Go(ex func()) (err error) {
	if ex == nil {
		return nil
	}

	// 单例初始化协程池：仅执行一次，初始化失败则后续直接降级
	antsOnce.Do(func() {
		antsPool, err = ants.NewPool(antsConfig.PoolSize)
		if err != nil {
			// 初始化失败日志：记录错误原因+使用的池大小
			hlog.Errorf("ants pool init failed, will use native go! [poolSize:%d] Err:%v", antsConfig.PoolSize, err)
			antsPool = nil // 显式置空，确保后续降级
			return
		}
		// 初始化成功日志
		hlog.Infof("ants pool init success! poolSize:%d", antsConfig.PoolSize)
	})

	// 初始化失败：直接降级为原生go执行，保证任务不丢失
	if err != nil || antsPool == nil {
		go ex()
		return nil
	}

	// 初始化成功：提交任务到协程池
	if submitErr := antsPool.Submit(ex); submitErr != nil {
		// 提交失败日志：记录原因，降级为原生go执行
		hlog.Errorf("ants pool submit task failed, fallback to native go Err:%v", submitErr)
		go ex()
	}
	return nil
}

/**
 * @description:关闭协程池（优雅退出）
 * @param {time.Duration} timeout 等待超时时间
 * @return {*}
 */
func CloseAntsPool(timeout time.Duration) error {
	if antsPool == nil {
		return nil
	}
	// 关闭协程池并记录日志
	err := antsPool.ReleaseTimeout(timeout)
	if err != nil {
		hlog.Errorf("ants pool close fail! Err:%v", err)
		return err
	}
	hlog.Info("ants pool close success")
	return nil
}
