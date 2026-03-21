/*
 * @Author: yeying
 * @Date: 2026-02-05 10:05:37
 * @FilePath: /dark_pkg/examples/config/local/main.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package main

import (
	"fmt"

	"github.com/ye-f-ying/dark_pkg/pkg/config"
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
	fmt.Println(base, conf)
	fmt.Println(conf.GetMysql())
	fmt.Println(config.GetGlobalAdapter[*BaseConfig, *CommonConfig]())
}
