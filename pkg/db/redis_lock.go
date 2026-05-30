/*
 * @Date: 2026-05-30 16:23:24
 * @LastEditTime: 2026-05-30 16:24:47
 * @FilePath: /dark_pkg/pkg/db/redis_lock.go
 * @Description:
 */
package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bsm/redislock"
	"github.com/redis/go-redis/v9"
)

var redisManage *RedisManage
var redisManageOnce sync.Once

func GetRedisManage() *RedisManage {
	redisManageOnce.Do(func() {
		redisCluster := GetRedisClient()
		redisManage = &RedisManage{redisCluster: redisCluster,
			redisLock: redislock.New(redisCluster),
			ctx:       GetRedisContext()}
	})
	return redisManage
}

/**
 * @description: 获取锁对象可以自由发挥锁的用法
 * @return {*}
 */
func GetRedisLock() *redislock.Client {
	return GetRedisManage().getRedisLock()
}

type RedisManage struct {
	redisCluster redis.UniversalClient
	redisLock    *redislock.Client
	ctx          context.Context
}

/**
 * @description: 在分布式锁中运行
 * @param {string} key 要锁定的key
 * @param {time.Duration} lockTTL 锁最长时间
 * @param {time.Duration} waitTimeout 逻辑执行超时时间
 * @param {func} lockFun 执行的加锁函数
 * @return {*}
 */
func (m *RedisManage) RunDistributedLocks(key string, lockTTL, waitTimeout time.Duration, lockFun func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	deadlineCtx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	//必须用新的context 不然cancel执行后可能会导致deadlineCtx被释放掉了 导致最后一个解锁失败
	ctx := context.Background()
	defer cancel()
	// 第一次立即尝试获取锁
	rl, err := TryLock(ctx, key, lockTTL)
	if err == nil && rl != nil {
		result, err := lockFun(ctx)
		rl.UnLock(ctx)
		return result, err
	}
	if err != redislock.ErrNotObtained {
		return nil, err //出现不是获取锁失败的错误
	}
	// 每 100ms 尝试一次获取锁，直到超时 其实就是忙等锁
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadlineCtx.Done(): //超时
			return ErrRedisLockTimeout, nil
		case <-ticker.C:
			newRl, err := TryLock(ctx, key, lockTTL)
			if err == nil {
				result, err := lockFun(ctx)
				newRl.UnLock(ctx)
				return result, err
			}
			if err != redislock.ErrNotObtained {
				return nil, err //出现不是获取锁失败的错误
			}
		}
	}
}

// redis分布式锁
type RedisLock struct {
	lock *redislock.Lock
}

/**
 * @description: 获取
 * @return {*}
 */
func (m *RedisLock) GetLock() *redislock.Lock {
	return m.lock
}

/**
 * @description:  对指定key 加锁
 * @param {context.Context} ctx
 * @param {string} key
 * @param {time.Duration} ttl
 * @return {*}
 */
func TryLock(ctx context.Context, key string, ttl time.Duration) (*RedisLock, error) {
	locker := GetRedisLock()
	l, err := locker.Obtain(ctx, fmt.Sprintf("lock:%s", key), ttl, nil)
	if err != nil {
		return nil, err
	}
	return &RedisLock{lock: l}, nil
}

/**
 * @description: 释放锁
 * @param {context.Context} ctx
 * @return {*}
 */
func (m *RedisLock) UnLock(ctx context.Context) error {
	if m.lock == nil {
		return nil
	}
	return m.lock.Release(ctx)
}

/**
 * @description: 刷新锁定时间
 * @param {context.Context} ctx
 * @param {time.Duration} ExlockTTL
 * @return {*}
 */
func (m *RedisLock) ExLockTime(ctx context.Context, ExlockTTL time.Duration) error {
	if m.lock == nil {
		return ErrServerIsNull
	}
	return m.lock.Refresh(ctx, ExlockTTL, nil)
}

/**
 * @description: 获取锁对象
 * @return {*}
 */
func (m *RedisManage) getRedisLock() *redislock.Client {
	return m.redisLock
}

/**
 * @description: 检查锁是否存在（不加锁）
 * @param {context.Context} ctx
 * @param {string} key
 * @return {bool} true=锁存在, false=锁不存在
 */
func CheckLockExists(ctx context.Context, key string) (bool, error) {
	// 拼接真实的锁 key
	lockKey := fmt.Sprintf("lock:%s", key)
	// 直接用 redis EXISTS 判断
	exists, err := GetRedisManage().redisCluster.Exists(ctx, lockKey).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

/**
 * @description: 获取锁
 * @param {context.Context} ctx
 * @param {string} key
 * @param {time.Duration} ttl
 * @param {*} wait
 * @param {time.Duration} frequency
 * @return {*}
 */
func AcquireLockWithWait(ctx context.Context, key string, ttl time.Duration, wait, frequency time.Duration) (*RedisLock, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()

	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			// 超时
			return nil, redislock.ErrNotObtained
		case <-ticker.C:
			redisLock, err := TryLock(ctx, key, ttl)
			if err == nil {
				return redisLock, nil
			}
			if err != redislock.ErrNotObtained {
				// 连接错误等
				return nil, err
			}
			// 否则继续重试
		}
	}
}
