/*
 * @Date: 2026-05-30 16:24:36
 * @LastEditTime: 2026-05-30 16:24:37
 * @FilePath: /dark_pkg/pkg/db/error.go
 * @Description:
 */
package db

import (
	"errors"
	"fmt"
)

var (
	//锁超时
	ErrRedisLockTimeout = fmt.Errorf("lock time out")

	ErrCahceDatas = fmt.Errorf("cache data error")

	ErrRedisError    = fmt.Errorf("redis error")
	ErrServerIsNull  = errors.New("is null")
	ErrQueryIdIsNull = errors.New("id is null")
)
