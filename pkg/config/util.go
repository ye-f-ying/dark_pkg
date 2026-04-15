/*
 * @Author: yeying
 * @Date: 2026-02-04 13:43:19
 * @FilePath: /dark_pkg/pkg/config/util.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

/**
 * @description: 读取本地配置文件到Viper（支持yaml）
 * @param {*viper.Viper} v
 * @param {string} path
 * @return {*}
 */
func LoadConfigFile(v *viper.Viper, path string) error {
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("读取文件[%s]失败：%w", path, err)
	}
	return nil
}

/**
 * @description: OverrideByEnv 环境变量覆盖Viper配置（键名：大写+下划线，如base.port → BASE_PORT）
 * @param {*viper.Viper} v
 * @return {*}
 */
func OverrideByEnv(v *viper.Viper) {
	v.AutomaticEnv()
	// 环境变量键名映射：将配置的点分隔转为下划线分隔（base.port → BASE_PORT）
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

/**
 * @description: ViperToStruct Viper配置绑定到结构体
 * @param {*viper.Viper} v
 * @param {any} dst
 * @return {*}
 */
func ViperToStruct(v *viper.Viper, dst any) error {
	if err := v.Unmarshal(dst); err != nil {
		return fmt.Errorf("配置绑定结构体失败：%w", err)
	}
	return nil
}

/**
 * @description: BytesToViper 字节数组转Viper（用于etcd读取的配置）
 * @param {[]byte} data
 * @return {*}
 */
func BytesToViper(data []byte) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBuffer(data)); err != nil {
		return nil, fmt.Errorf("字节数组转Viper失败：%w", err)
	}
	return v, nil
}

/**
 * @description: NewEtcdClient 创建etcd客户端
 * @param {*ConfigOptions} opts
 * @return {*}
 */
func NewEtcdClient(opts *ConfigOptions) (*clientv3.Client, error) {
	return clientv3.New(clientv3.Config{
		Endpoints:   opts.EtcdAddrs,
		DialTimeout: time.Duration(opts.EtcdTimeout) * time.Second,
		Username:    opts.EtcdUser,
		Password:    opts.EtcdPwd,
	})
}

// EtcdPut 将字节数据写入etcd指定Key
func EtcdPut(client *clientv3.Client, key string, data []byte, timeout int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	_, err := client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("etcd Put[%s]失败：%w", key, err)
	}
	return nil
}

// EtcdGet 从etcd读取指定Key的字节数据
func EtcdGet(client *clientv3.Client, key string, timeout int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	resp, err := client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("etcd Get[%s]失败：%w", key, err)
	}
	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("etcd中无Key[%s]的配置", key)
	}
	return resp.Kvs[0].Value, nil
}

// ReadLocalFile 读取本地文件为字节数组（用于读取common.yml写入etcd）
func ReadLocalFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取本地文件[%s]失败：%w", path, err)
	}
	return data, nil
}

/**
 * @description: 合成远程+本地配置
 * @param {*viper.Viper} localViper 本地配置
 * @param {*viper.Viper} remoteViper 远程配置
 * @param {[]string} protectKeys 需要保护的数据
 * @return {*}
 */
func MergeRemoteToLocalSafely(localViper *viper.Viper, remoteViper *viper.Viper, protectKeys []string) error {
	// 默认使用全局 viper
	if localViper == nil {
		return fmt.Errorf("local viper is not nil")
	}
	if remoteViper == nil {
		return fmt.Errorf("remote viper is not nil")
	}

	// 把需要保护的键存为 map，快速判断
	protectMap := make(map[string]struct{}, len(protectKeys))
	for _, k := range protectKeys {
		protectMap[k] = struct{}{}
	}

	// 获取远程所有配置项
	remoteAllKeys := remoteViper.AllKeys()

	// 遍历远程配置 → 合并到本地，但跳过保护键
	for _, key := range remoteAllKeys {
		// 本地保护的字段（serverID/编号），直接跳过，不覆盖
		if _, isProtected := protectMap[key]; isProtected {
			continue
		}

		// 非保护字段 → 用远程配置覆盖本地
		val := remoteViper.Get(key)
		localViper.Set(key, val)
	}

	return nil
}
