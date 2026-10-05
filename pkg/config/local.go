/*
 * @Date: 2026-04-15 15:54:46
 * @LastEditTime: 2026-10-05 14:47:53
 * @FilePath: /dark_pkg/pkg/config/local.go
 * @Description:
 */
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// LocalConfig 纯本地模式泛型适配器：T=基础配置，U=公共配置
type LocalConfig[T IConfig] struct {
	opts ConfigOptions // 配置选项（去掉指针，避免空指针）
	cfg  *viper.Viper  // 最终配置
}

// Init 初始化本地模式：1.加载基础配置 2.命令+环境覆盖 3.加载本地公共配置 4.合并
func (m *LocalConfig[T]) Init() error {
	// 加载本地基础配置（app.yml）
	if m.cfg != nil {
		return nil // adapter 已加载覆盖过，直接复用
	}
	initFlag()
	if cliConfigPath != "" {
		m.opts.ConfigPath = cliConfigPath // 覆盖要写回 m.opts，原来只改了局部 optsCopy
	}
	if cliConfigWrite != -1 {
		m.opts.IsWrite = cliConfigWrite == 1
	}
	v := viper.New()
	if err := LoadConfigFile(v, m.opts.ConfigPath); err != nil {
		return err
	}
	OverrideByEnv(v) // 直接作用在生效配置上，原来作用在被丢弃的 cfgViper 上
	localFlag(v)
	m.cfg = v
	return nil
}

// GetConfig 泛型版：获取分离的基础配置和公共配置
func (m *LocalConfig[T]) GetConfig() (T, error) {
	var base T
	// 分别绑定基础/公共配置到业务自定义的泛型结构体
	if err := ViperToStruct(m.cfg, &base); err != nil {
		return base, fmt.Errorf("绑定基础配置失败：%w", err)
	}
	return base, nil
}

// Watch 本地模式泛型版：空实现（无配置变化），回调不触发
func (l *LocalConfig[T]) Watch(callback func(newT T, err error)) {}
