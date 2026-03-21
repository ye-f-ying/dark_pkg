/*
 * @Author: yeying
 * @Date: 2026-02-05 13:35:30
 * @FilePath: /dark_pkg/pkg/kitex_etcd/kitex_client_manager.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package kitex_etcd

import (
	"context"
	"fmt"
	"net"
	"reflect"

	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/discovery"
	"github.com/cloudwego/kitex/pkg/registry"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"
	"github.com/ye-f-ying/dark_pkg/pkg/config"
)

// -----------------------------
// ClientManager 单例
// -----------------------------
var (
	clientManager     *ClientManager
	clientManagerOnce sync.Once
)

// 获取客户端管理器
func GetKitexClientManager() *ClientManager {
	clientManagerOnce.Do(func() {
		clientManager = &ClientManager{}
		err := clientManager.Init()
		if err != nil {
			hlog.Errorf("[KitexClientManager] 初始化失败: %v", err)
			panic(err)
		}
	})
	return clientManager
}

// -----------------------------
// ClientFactory（构造 Kitex client）
// -----------------------------

type ClientFactory func(opt ...client.Option) (interface{}, error)

// service → type → factory
var globalFactories sync.Map // map[string]*sync.Map

// 注册 client 工厂
func RegisterKitexClient[T any](service string, f ClientFactory) {
	typeName := typeNameOf[T]()

	val, _ := globalFactories.LoadOrStore(service, &sync.Map{})
	m := val.(*sync.Map)
	m.Store(typeName, f)
}

// -----------------------------
// ClientManager 结构体
// -----------------------------
type ClientManager struct {
	registry  discovery.Resolver
	initOnce  sync.Once
	clientMap sync.Map // service::type -> instance
	mu        sync.Mutex
}

/**
 * @description: 初始化
 * @return {*}
 */
func (m *ClientManager) Init() error {
	var err error
	m.initOnce.Do(func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		gaCfg, _ := config.GetGlobalAdapter[config.IBaseConfig, config.ICommonConfig]()
		if gaCfg == nil {
			err = fmt.Errorf("kitex etcd not configured")
			return
		}
		baseCfg, _, _ := gaCfg.GetConfig()
		if baseCfg == nil || baseCfg.GetEtcdConfig() == nil {
			err = fmt.Errorf("kitex etcd not configured")
			return
		}
		cfg := baseCfg.GetEtcdConfig()
		etcdCfg = cfg
		timeout := cfg.DialTimeout
		if timeout <= 0 {
			timeout = 3
		}
		m.registry, err = etcd.NewEtcdResolver(cfg.Endpoints,
			etcd.WithDialTimeoutOpt(time.Duration(timeout)*time.Second),
			etcd.WithAuthOpt(cfg.Username, cfg.Password),
			WithDialKeepAliveTime(cfg.DialKeepAliveTime),
			WithDialKeepAliveTimeout(cfg.DialKeepAliveTimeout),
		)
		if err != nil {
			return
		}
	})

	return err
}

/**
 * @description: 泛型获取 Kitex 客户端
 * @return {*}
 */
func GetKitexClientCtx[T any](ctx context.Context, service string) (T, error) {
	var zero T

	cm := GetKitexClientManager()
	typeName := typeNameOf[T]()
	key := service + "::" + typeName

	// fast path: 已缓存
	if v, ok := cm.clientMap.Load(key); ok {
		if cli, ok2 := v.(T); ok2 {
			return cli, nil
		}
	}

	// 找到 factory
	val, ok := globalFactories.Load(service)
	if !ok {
		return zero, fmt.Errorf("service=%s 未注册", service)
	}
	m := val.(*sync.Map)

	f, ok := m.Load(typeName)
	if !ok {
		return zero, fmt.Errorf("service=%s 未注册类型=%s", service, typeName)
	}

	factory := f.(ClientFactory)

	// TODO: 这里按需要添加参数
	tempClient, err := factory(client.WithResolver(cm.registry), client.WithSuite(tracing.NewClientSuite())) // client.WithTracer(NewTracer())
	if err != nil {
		return zero, err
	}
	if clientInstance, ok := tempClient.(T); ok {
		cm.clientMap.Store(key, clientInstance)
		return clientInstance, nil
	}
	return zero, fmt.Errorf("type mismatch")
}

/**
 * @description: 类型反射
 * @return {*}
 */
func typeNameOf[T any]() string {
	typ := reflect.TypeOf((*T)(nil)).Elem()
	return typ.PkgPath() + "." + typ.Name()
}

var (
	once         sync.Once
	etcdRegistry registry.Registry
)

/**
 * @description: 获取服务端使用的EtcdRegistry
 * @return {*}
 */
func GetDefaultEtcdRegistry() registry.Registry {
	once.Do(func() {
		cfg := etcdCfg
		if cfg == nil || len(cfg.Endpoints) == 0 {
			hlog.Error("etcd config is nil or empty endpoints")
			panic("etcd config is nil or empty endpoints")
		}
		timeout := time.Duration(cfg.DialTimeout)
		if timeout <= 0 {
			timeout = 3
		}
		var etcdError error = nil
		etcdRegistry, etcdError = etcd.NewEtcdRegistry(cfg.Endpoints,
			etcd.WithDialTimeoutOpt(time.Duration(timeout)*time.Second),
			etcd.WithAuthOpt(cfg.Username, cfg.Password),
			WithDialKeepAliveTime(cfg.DialKeepAliveTime),
			WithDialKeepAliveTimeout(cfg.DialKeepAliveTimeout),
		)
		if etcdError != nil {
			hlog.Errorf("etcd Registry 创建异常！%v", etcdError)
			panic(etcdError)
		}
	})

	return etcdRegistry
}

/**
 * @description: 设置DialKeepAliveTime
 * @param {int} dialKeepAliveTime
 * @return {*}
 */
func WithDialKeepAliveTime(dialKeepAliveTime int) etcd.Option {
	return func(cfg *etcd.Config) {
		if dialKeepAliveTime <= 0 {
			return
		}
		cfg.EtcdConfig.DialKeepAliveTime = time.Duration(dialKeepAliveTime) * time.Second
	}
}

/**
 * @description: 设置 DialKeepAliveTimeout
 * @param {int} dialKeepAliveTimeout
 * @return {*}
 */
func WithDialKeepAliveTimeout(dialKeepAliveTimeout int) etcd.Option {
	return func(cfg *etcd.Config) {
		if dialKeepAliveTimeout <= 0 {
			return
		}
		cfg.EtcdConfig.DialKeepAliveTimeout = time.Duration(dialKeepAliveTimeout) * time.Second
	}
}

func GetDefaultServerOption(serverName string) []server.Option {
	cfg := etcdCfg
	if cfg == nil {
		hlog.Error("没有配置Kitex！")
		panic("没有配置Kitex！")
	}
	var options []server.Option
	r := GetDefaultEtcdRegistry()
	if cfg.Address == "" {
		hlog.Error("没有配置Kitex地址及端口！")
		panic("没有配置Kitex地址及端口！")
	}
	addr, err := net.ResolveTCPAddr("tcp", cfg.Address)
	if err != nil {
		hlog.Errorf("启动监听出现异常！%v", err)
		panic(err)
	}

	options = append(options, server.WithRegistry(r),
		server.WithServiceAddr(addr),
		server.WithSuite(tracing.NewServerSuite()),
		//server.WithTracer(NewTracer()),
		server.WithMuxTransport(), // 连接多路复用
		server.WithServerBasicInfo(
			&rpcinfo.EndpointBasicInfo{
				ServiceName: serverName,
			}))

	return options

}
