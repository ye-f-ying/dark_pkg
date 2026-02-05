/*
 * @Author: yeying
 * @Date: 2026-02-04 13:03:07
 * @FilePath: /dark_pkg/pkg/config/local.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// LocalConfig 纯本地模式泛型适配器：T=基础配置，U=公共配置
type LocalConfig[T IBaseConfig, U ICommonConfig] struct {
	opts      ConfigOptions  // 配置选项（去掉指针，避免空指针）
	v         *viper.Viper   // 	综合配置
	base      *viper.Viper   // 	基础配置
	cfg       *viper.Viper   // 	公共配置
	cmdParams map[string]any // 命令行参数（用于覆盖基础配置）
}

// Init 初始化本地模式：1.加载基础配置 2.命令+环境覆盖 3.加载本地公共配置 4.合并
func (m *LocalConfig[T, U]) Init() error {
	// 步骤1：加载本地基础配置（app.yml）
	m.base = viper.New()
	if err := LoadConfigFile(m.base, m.opts.AppConfigPath); err != nil {
		return err
	}

	// 步骤2：按优先级覆盖基础配置：命令参数 > 环境变量 > 本地配置
	OverrideByEnv(m.base) // 环境变量覆盖
	if m.cmdParams != nil {
		OverrideByCmd(m.base, m.cmdParams) // 命令参数覆盖
	}

	// 步骤3：加载本地公共配置（config.yml）
	m.cfg = viper.New()
	if err := LoadConfigFile(m.cfg, m.opts.CommonConfigPath); err != nil {
		return err
	}

	// 步骤4：合并基础配置+公共配置为总配置
	m.v = MergeViper(m.base, m.cfg)
	return nil
}

// GetConfig 泛型版：获取分离的基础配置和公共配置
func (l *LocalConfig[T, U]) GetConfig() (T, U, error) {
	var base T
	var common U
	// 分别绑定基础/公共配置到业务自定义的泛型结构体
	if err := ViperToStruct(l.base, &base); err != nil {
		return base, common, fmt.Errorf("绑定基础配置失败：%w", err)
	}
	if err := ViperToStruct(l.cfg, &common); err != nil {
		return base, common, fmt.Errorf("绑定公共配置失败：%w", err)
	}
	return base, common, nil
}

// Watch 本地模式泛型版：空实现（无配置变化），回调不触发
func (l *LocalConfig[T, U]) Watch(callback func(newT T, newU U, err error)) {}

// SetCmdParams 设置命令行参数（供外部传入，用于覆盖基础配置）
func (l *LocalConfig[T, U]) SetCmdParams(params map[string]any) {
	l.cmdParams = params
}
