/*
 * @Date: 2026-04-15 15:54:46
 * @LastEditTime: 2026-04-15 16:57:23
 * @FilePath: /dark_pkg/pkg/config/local.go
 * @Description:
 */
package config

import (
	"fmt"

	"github.com/cloudwego/hertz/pkg/common/hlog"
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
	if m.cfg == nil {
		m.cfg = viper.New()
		if err := LoadConfigFile(m.cfg, m.opts.ConfigPath); err != nil {
			return err
		}

		// 合并默认选项和传入选项
		optsCopy := *defaultOpts
		// 初始化命令行参数
		initFlag()
		if cliConfigPath != "" {
			optsCopy.ConfigPath = cliConfigPath
		}
		if cliConfigWrite != -1 {
			if cliConfigWrite == 1 {
				optsCopy.IsWrite = true
			} else {
				optsCopy.IsWrite = false
			}
		}
		// 初始化viper，加载本地基础配置文件-命令参数>环境变量>本地配置 覆盖，获取有效配置
		cfgViper := viper.New()
		cfgViper.SetConfigFile(optsCopy.ConfigPath)
		cfgViper.SetConfigType("yaml")
		if loadErr := cfgViper.ReadInConfig(); loadErr != nil {
			/*err = fmt.Errorf("预加载本地基础配置失败：%w", loadErr)
			return*/
			hlog.Warnf("预加载本地基础配置失败：%v 使用默认配置！", loadErr)
			defaultConfig(cfgViper)
		}
		// 环境变量覆盖：自动映射，点分隔转下划线（如etcd_config.addrs → ETCD_CONFIG_ADDRS）
		OverrideByEnv(cfgViper)

		// 命令参数覆盖
		localFlag(cfgViper)

	}
	return nil
}

// GetConfig 泛型版：获取分离的基础配置和公共配置
func (l *LocalConfig[T]) GetConfig() (T, error) {
	var base T
	// 分别绑定基础/公共配置到业务自定义的泛型结构体
	if err := ViperToStruct(l.cfg, &base); err != nil {
		return base, fmt.Errorf("绑定基础配置失败：%w", err)
	}
	return base, nil
}

// Watch 本地模式泛型版：空实现（无配置变化），回调不触发
func (l *LocalConfig[T]) Watch(callback func(newT T, err error)) {}
