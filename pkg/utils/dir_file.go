/*
 * @Author: yeying
 * @Date: 2026-02-05 12:17:32
 * @FilePath: /dark_pkg/pkg/utils/dir_file.go
 * @Description:
 *
 * Copyright (c) 2026 by yeying, All Rights Reserved.
 */
package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 文件夹权限
const DirPerm = 0755

/**
 * @description: 判断文件夹是否存在（自动兼容相对/绝对路径，路径自动清理，修复权限错误导致的误判）
 * @param {string} dir 任意路径（相对如./logs、绝对如/usr/local/logs均可）
 * @return {bool} 仅当「路径存在且是文件夹」时返回true，其余情况（不存在/是文件/有错误）均返回false
 */
func IsDirExist(dir string) bool {
	// 1. 清理路径：去空格、标准化分隔符，兼容相对/绝对路径
	cleanDir := filepath.Clean(strings.TrimSpace(dir))
	// 2. 获取文件状态，有任何错误直接返回false
	info, err := os.Stat(cleanDir)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("[WARN] 检查目录状态失败：路径=%s，错误=%v\n", cleanDir, err)
		}
		return false
	}
	// 3. 无错误时，仅当是文件夹才返回true，是文件则返回false
	return info.IsDir()
}

/**
 * @description: 文件夹不存在则创建（递归创建多级目录，自动兼容相对/绝对路径，路径自动清理）
 * @param {string} dir 任意路径（相对如./logs/2026、绝对如D:/app/logs均可）
 * @return {error} 创建成功/已存在返回nil，创建失败返回具体错误
 */
func MkdirIfNotExist(dir string) error {
	// 复用清理后的路径，保证判断和创建的路径一致
	cleanDir := filepath.Clean(strings.TrimSpace(dir))
	// 路径为空则直接返回错误（避免创建空路径导致的异常）
	if cleanDir == "" {
		return os.ErrInvalid
	}
	// 文件夹已存在则直接返回
	if IsDirExist(cleanDir) {
		return nil
	}
	// 原生支持相对/绝对路径的递归创建
	return os.MkdirAll(cleanDir, DirPerm)
}
