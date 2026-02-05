/*
 * @Author: yeying
 * @Date: 2026-02-05 12:08:56
 * @FilePath: /dark_pkg/examples/log/main.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package main

import (
	"fmt"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/ye-f-ying/dark_pkg/pkg/config"
	"github.com/ye-f-ying/dark_pkg/pkg/zap"
)

type BaseConfig struct {
	config.BaseConfig `mapstructure:",squash"` // 需要扁平化
}

type CommonConfig struct {
	config.CommonConfig `mapstructure:",squash"`
}

func main() {
	cfg, err := config.Init[*BaseConfig, *CommonConfig]()
	if err != nil {
		fmt.Println(err)
		return
	}
	base, conf, err := cfg.GetConfig()
	if err != nil {
		hlog.Errorf("读取配置文件失败！%v", err)
		return
	}
	zap.InitZap(base, conf)
	hlog.Debug("debug")
	hlog.Info("Info")
	hlog.Error("Error")
	hlog.Warn("Warn")

}
