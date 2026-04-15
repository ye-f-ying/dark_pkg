/*
 * @Author: yeying
 * @Date: 2026-02-05 10:16:53
 * @FilePath: /dark_pkg/examples/config/etcd/main.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package main

import (
	"fmt"

	"github.com/ye-f-ying/dark_pkg/pkg/config"
)

type Config struct {
	config.DefaultConfig `mapstructure:",squash"` // 需要扁平化
	TestCfg              string                   `mapstructure:"test_cfg"  default:"test"` // 分布式ID
}

func main() {
	//cfg, err := config.Init[*BaseConfig, *CommonConfig](config.WithIsWrite(true))
	cfg, err := config.Init[*Config]()
	if err != nil {
		fmt.Println(err)
		return
	}
	conf, err := cfg.GetConfig()
	fmt.Println(conf)
	fmt.Println(conf.GetMysql())
	globalAdapter, err := config.GetGlobalAdapter[*Config]()
	gConf, err := globalAdapter.GetConfig()
	fmt.Println(gConf.GetServerID())
	fmt.Println(gConf.GetMysql())
	fmt.Println(gConf.TestCfg)
}
