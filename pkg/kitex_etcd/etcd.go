/*
 * @Author: yeying
 * @Date: 2026-02-05 13:33:59
 * @FilePath: /dark_pkg/pkg/kitex_etcd/etcd.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package kitex_etcd

import (
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/ye-f-ying/dark_pkg/pkg/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	etcdDefault *clientv3.Client
	etcdOnce    sync.Once
	etcdCfg     *config.ETCDConfig
)

/**
 * @description: 默认的etcd客户端 -- kratos 服务端和客户端使用单独的 etcd 不与这个共用 这个用于单独ETCD 相关操作 以隔离业务与grpc操作互不影响
 * @return {*}
 */
func GetDefaultETCDClient() *clientv3.Client {
	etcdOnce.Do(func() {
		var err error
		etcdDefault, err = NewETCDclient()
		if err != nil {
			panic(err)
		}
	})

	return etcdDefault
}

/**
 * @description: 新建一个ETCD客户端
 * @return {*}
 */
func NewETCDclient() (*clientv3.Client, error) {
	cfg := etcdCfg
	if cfg == nil {
		return nil, fmt.Errorf("kratos etcd not config")
	}
	dialTimeout := cfg.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 3 //默认3秒
	}
	dialKeepAliveTime := cfg.DialKeepAliveTime
	if dialKeepAliveTime <= 0 {
		dialKeepAliveTime = 30
	}
	dialKeepAliveTimeout := cfg.DialKeepAliveTimeout
	if dialKeepAliveTimeout <= 0 {
		dialKeepAliveTimeout = 20
	}
	etcdConfig := clientv3.Config{
		Endpoints:            cfg.Endpoints,
		DialTimeout:          time.Duration(dialTimeout) * time.Second,
		Username:             cfg.Username, // etcd 用户名
		Password:             cfg.Password, // etcd 密码
		DialKeepAliveTime:    time.Duration(dialKeepAliveTime) * time.Second,
		DialKeepAliveTimeout: time.Duration(dialKeepAliveTimeout) * time.Second,
		//Logger:               szap.GetLogger().Logger(),
	}
	client, err := clientv3.New(etcdConfig)
	if err != nil {
		hlog.Errorf("[pkg/kitex_etcd/etcd]-[NewETCDclient] 连接ETCD出现错误：%v", err)
		return nil, err
	}
	return client, nil
}
