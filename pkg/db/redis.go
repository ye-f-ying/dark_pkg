/*
 * @Date: 2026-05-30 16:16:02
 * @LastEditTime: 2026-05-31 15:43:36
 * @FilePath: /dark_pkg/pkg/db/redis.go
 * @Description:
 */
package db

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/redis/go-redis/v9"
	"github.com/ye-f-ying/dark_pkg/pkg/config"
)

var (
	configRedis      *config.REDIS
	redisCluster     RedisCluster
	redisClusterOnce sync.Once
)

/**
 * @description: 获取redis配置
 * @return {*}
 */
func GetConfigRedis() *config.REDIS {
	if configRedis == nil {
		gaCfg := config.GetGlobalConfig()
		if gaCfg == nil {
			panic("redis not configured")
		}
		cfg := gaCfg.GetRedis()
		if cfg == nil {
			panic("redis not configured")
		}
		configRedis = cfg
	}
	return configRedis
}

// redis集群连接
type RedisCluster struct {
	universalClient redis.UniversalClient
	ctx             context.Context
}

/**
 * @description: 初始化redis连接器
 * @return {*}
 */
func (m *RedisCluster) Init() error {
	cfg := GetConfigRedis()
	m.ctx = context.Background()
	redisLen := len(cfg.Address)
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 10 * runtime.GOMAXPROCS(0)
	}
	if redisLen <= 0 {
		return fmt.Errorf("连接配置异常！redis地址必须大于1个")
	} else if redisLen == 1 { //单机连接
		m.universalClient = redis.NewClient(&redis.Options{
			Addr:     cfg.Address[0],
			DB:       cfg.DB,
			Username: cfg.UserName,
			Password: cfg.Password, // 无密码则为空

			PoolSize:     cfg.PoolSize,     // 连接池大小
			MinIdleConns: cfg.MinIdleConns, // 最小空闲连接
			ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Millisecond,
			WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Millisecond,
			//DisableTracking: true,
		})
	} else { //集群连接
		m.universalClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs: cfg.Address,
			//DB:           cfg.Redis.DB,
			Username: cfg.UserName,
			Password: cfg.Password, // 无密码则为空
			PoolSize: cfg.PoolSize, // 连接池大小

			MinIdleConns: cfg.MinIdleConns, // 最小空闲连接
			ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Millisecond,
			WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Millisecond,
			OnConnect: func(ctx context.Context, cn *redis.Conn) error {
				hlog.Infof("连接redis:%s", cn.String())
				return nil
			},
		})
	}

	// 测试连接
	if err := m.universalClient.Ping(m.ctx).Err(); err != nil {
		hlog.Errorf("连接Redis出现错误：%s", err.Error())
		return err
	}
	return nil
}

/**
 * @description: 获取redis连接
 * @return {*}
 */
func (m *RedisCluster) GetRedis() redis.UniversalClient {
	return m.universalClient
}

/**
 * @description: 获取默认的Context
 * @return {*}
 */
func (m *RedisCluster) GetContext() context.Context {
	return m.ctx
}

/**
 * @description: 初始化redis连接
 * @return {*}
 */
func InitRedis() (err error) {
	redisClusterOnce.Do(func() {
		redisCluster = RedisCluster{}
		err = redisCluster.Init()
	})
	return err
}

/**
 * @description: 获取redis连接
 * @return {*}
 */
func GetRedisClient() redis.UniversalClient {
	return redisCluster.GetRedis()
}

/**
 * @description: 获取默认的Context
 * @return {*}
 */
func GetRedisContext() context.Context {
	if redisCluster.universalClient == nil {
		InitRedis()
	}
	return redisCluster.GetContext()
}

/**
 * @description: redis 频率限制
 * @param {context.Context} ctx
 * @param {string} key
 * @param {int} limitMax
 * @param {time.Duration} limitTime
 * @return {*}
 */
func FrequencyCheckRedis(ctx context.Context, key string, limitMax int, limitTime time.Duration) (isAllow bool, count int, err error) {
	pipe := GetRedisClient().Pipeline()

	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, limitTime) // 每次刷新时间 使用滚动限制

	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}

	count = int(incr.Val())
	isAllow = count <= limitMax

	return isAllow, count, nil
}
