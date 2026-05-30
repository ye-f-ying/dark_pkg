/*
 * @Date: 2026-05-30 16:27:46
 * @LastEditTime: 2026-05-30 16:28:47
 * @FilePath: /dark_pkg/pkg/cache/redis_cache.go
 * @Description:
 */
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"

	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ye-f-ying/dark_pkg/pkg/db"
)

// 基础过期时间
const REDIC_CACHE_BADE_EX = 60 * 5

// 最小过期时间
const REDIC_CACHE_MIN_EX = 60

type RedisParam struct {
	Node    string          // hash 时为缓存的key
	Key     string          // 缓存的key hash时为列表的key
	MaxTime int64           // 最大过期时间
	IsCache bool            // 是否缓存
	Ctx     context.Context // 透传Context
}

type CacheData[T any] struct {
	Data *T    `json:"data"`
	Time int64 `json:"time"`
}

type CacheDataGeneral[T any] struct {
	Data       T     `json:"data"`
	Time       int64 `json:"time"`
	ExpireTime int64 `json:"expire_time"` // 过期时间点
}

/**
 * @description: redis 缓存
 * @return {*}
 */
func RedisCache[T any](param RedisParam, fetchFunc func() (*T, error)) (*T, error) {
	key := param.Key
	if param.Ctx == nil {
		param.Ctx = context.Background()
	}

	var cacheTime time.Duration
	if param.MaxTime > 0 {
		cacheTime = ExpireWithJitter(param.MaxTime)
	} else {
		cacheTime = ExpireWithJitter(REDIC_CACHE_BADE_EX)
	}

	ctx := db.GetRedisContext()
	if param.IsCache {
		// 读取缓存
		cacheStr, err := db.GetRedisClient().Get(ctx, key).Result()
		if err == nil {
			if cacheStr == "" {
				return nil, nil
			}
			var cd CacheData[T]
			if err = json.Unmarshal([]byte(cacheStr), &cd); err == nil {
				return cd.Data, nil
			}
			return nil, err
		}
		if !errors.Is(err, redis.Nil) {
			return nil, err
		}
	}

	// 执行源函数
	res, errFun := fetchFunc()
	if errFun != nil {
		return nil, errFun
	}

	cd := CacheData[T]{Data: res, Time: time.Now().Unix()}
	jsonBytes, err := json.Marshal(cd)
	if err != nil {
		return nil, err
	}
	err = db.GetRedisClient().Set(ctx, key, jsonBytes, cacheTime).Err()
	if err != nil {
		return nil, err
	}

	return res, nil
}

/**
 * @description: 完全匹配 通用缓存
 * @return {*}
 */
func RedisCacheGeneral[T any](param RedisParam, fetchFunc func() (T, error)) (T, error) {
	key := param.Key
	if param.Ctx == nil {
		param.Ctx = context.Background()
	}
	var cacheTime time.Duration
	if param.MaxTime > 0 {
		cacheTime = ExpireWithJitter(param.MaxTime)
	} else {
		cacheTime = ExpireWithJitter(REDIC_CACHE_BADE_EX)
	}
	ctx := db.GetRedisContext()
	var tempT T
	if param.IsCache {
		// 读取缓存
		cacheStr, err := db.GetRedisClient().Get(ctx, key).Result()
		if err == nil {
			if cacheStr == "" {
				return tempT, nil
			}
			var cd CacheDataGeneral[T]
			if err = json.Unmarshal([]byte(cacheStr), &cd); err == nil {
				return cd.Data, nil
			}
			return tempT, err
		}
		if !errors.Is(err, redis.Nil) {
			return tempT, err
		}
	}

	//这里后面可以根据量加上redis锁或者其他逻辑 防止并发穿透
	// 执行源函数
	res, errFun := fetchFunc()
	if errFun != nil {
		return tempT, errFun
	}

	cd := CacheDataGeneral[T]{Data: res, Time: time.Now().Unix()}
	jsonBytes, err := json.Marshal(cd)
	if err != nil {
		return tempT, err
	}
	err = db.GetRedisClient().Set(ctx, key, jsonBytes, cacheTime).Err()
	if err != nil {
		return tempT, err
	}

	return res, nil
}

/**
 * @description: 返回一个带随机错位的过期时间，确保最小为1分钟
 * @param {int64} maxSeconds //最大过期时间
 * @return {*}
 */
func ExpireWithJitter(maxSeconds int64) time.Duration {
	// 生成一个 [-jitterSeconds, jitterSeconds] 的随机数
	offset := rand.Int63n(maxSeconds*2+1) - maxSeconds
	expire := REDIC_CACHE_BADE_EX + offset

	// 确保不小于 60 秒
	if expire < REDIC_CACHE_MIN_EX {
		expire = REDIC_CACHE_MIN_EX + rand.Int63n(REDIC_CACHE_MIN_EX)
	}
	return time.Duration(expire) * time.Second
}

/**
 * @description: redis hash 缓存
 * @return {*}
 */
func RedisCacheHash[T any](param RedisParam, fetchFunc func() (T, error)) (T, error) {
	var (
		hashKey = param.Node
		field   = param.Key
		ctx     = db.GetRedisContext()
		tempT   T
		client  = db.GetRedisClient()
	)
	if param.Ctx == nil {
		param.Ctx = context.Background()
	}

	// ===============================
	// 读取缓存逻辑
	// ===============================
	if param.IsCache {
		cacheStr, err := client.HGet(ctx, hashKey, field).Result()
		if err == nil && cacheStr != "" {
			var cd CacheDataGeneral[T]
			if err = json.Unmarshal([]byte(cacheStr), &cd); err == nil {
				now := time.Now().Unix()
				if now < cd.ExpireTime {
					// 自动延长 hashKey 的 TTL （访问越频繁，越不会过期）
					ttl, _ := client.TTL(ctx, hashKey).Result()
					if ttl > 0 && ttl < (time.Hour*24) { // 剩余 < 1 天自动续期
						newTTL := ExpireWithJitter(param.MaxTime)
						_ = client.Expire(ctx, hashKey, newTTL)
					}
					return cd.Data, nil
				}
				// 否则过期，走刷新逻辑
			} else {
				return tempT, err
			}
		} else if err != nil && !errors.Is(err, redis.Nil) {
			return tempT, err
		}
	}

	// ===============================
	// 缓存未命中或过期 → 调用源函数
	// ===============================
	res, errFun := fetchFunc()
	if errFun != nil {
		return tempT, errFun
	}

	var baseTTL time.Duration
	if param.MaxTime > 0 {
		baseTTL = ExpireWithJitter(param.MaxTime)
	} else {
		baseTTL = ExpireWithJitter(REDIC_CACHE_BADE_EX)
	}

	cd := CacheDataGeneral[T]{
		Data:       res,
		Time:       time.Now().Unix(),
		ExpireTime: time.Now().Add(baseTTL).Unix(),
	}

	jsonBytes, err := json.Marshal(cd)
	if err != nil {
		return tempT, err
	}

	// ===============================
	// Pipeline 写入缓存（集群安全）
	// ===============================
	exists, _ := client.Exists(ctx, hashKey).Result()
	pipe := client.TxPipeline() // 在集群中安全的 pipeline
	pipe.HSet(ctx, hashKey, field, jsonBytes)
	if exists == 0 {
		pipe.Expire(ctx, hashKey, baseTTL)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		// pipeline 出错（如集群CROSSSLOT等）→ fallback
		if err := client.HSet(ctx, hashKey, field, jsonBytes).Err(); err != nil {
			return tempT, err
		}
		if exists == 0 {
			_ = client.Expire(ctx, hashKey, baseTTL)
		}
	}

	return res, nil
}

/**
 * @description: hash 值清除
 * @param {RedisParam} param
 * @param {bool} delAll
 * @return {*}
 */
func RedisCacheHashDelete(node, key string) error {
	ctx := db.GetRedisContext()
	client := db.GetRedisClient()

	err := client.HDel(ctx, node, key).Err()
	if err != nil {
		return err
	}
	return nil
}
