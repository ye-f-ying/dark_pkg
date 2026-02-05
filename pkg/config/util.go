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
 * @description: OverrideByCmd 命令行参数覆盖Viper配置（通过cobra传入的参数映射）
 * @param {*viper.Viper} v
 * @param {map[string]any} cmdParams
 * @return {*}
 */
func OverrideByCmd(v *viper.Viper, cmdParams map[string]any) {
	for key, val := range cmdParams {
		if val != nil {
			v.Set(key, val)
		}
	}
}

/**
 * @description: MergeViper 合并两个Viper配置：src覆盖dst同名键（递归，支持嵌套）
 * @param {*} dst
 * @param {*viper.Viper} src
 * @return {*}
 */
func MergeViper(dst, src *viper.Viper) *viper.Viper {
	sum := viper.New()
	for _, key := range src.AllKeys() {
		sum.Set(key, src.Get(key))
	}
	for _, key := range dst.AllKeys() {
		sum.Set(key, dst.Get(key))
	}
	return sum
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
