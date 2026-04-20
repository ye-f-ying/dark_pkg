/*
 * @Date: 2026-04-20 17:16:55
 * @LastEditTime: 2026-04-20 18:41:09
 * @FilePath: /dark_pkg/examples/nats/main.go
 * @Description:
 */
package main

import (
	"fmt"
	"time"

	"github.com/ye-f-ying/dark_pkg/pkg/config"
	"github.com/ye-f-ying/dark_pkg/pkg/messagemgr"
)

type Config struct {
	config.DefaultConfig `mapstructure:",squash"` // 需要扁平化
}

func main() {
	_, err := config.Init[*Config]()
	if err != nil {
		fmt.Println(err)
		return
	}
	mgr := messagemgr.GetDefaultManager()
	mgr.RegisterHandler(&messagemgr.Handler{
		Transport: messagemgr.QueueTransport,
		BizType:   "test",
		Handle: func(msg *messagemgr.Message) error {
			fmt.Println(string(msg.Data))
			return nil
		},
		Options: messagemgr.HandlerOptions{
			MaxRetry:    3,                               //重试3次
			AckWait:     time.Second * time.Duration(10), //最多等待10秒
			Concurrency: 100,                             //最多等待条未确认的消息
		},
	})

	mgr.RegisterHandler(&messagemgr.Handler{
		Transport: messagemgr.BroadcastTransport,
		BizType:   "BroadcastTest",
		Handle: func(msg *messagemgr.Message) error {
			fmt.Println(string(msg.Data))
			if msg.GetRetryCount() >= msg.GetRetryMax() {
				return fmt.Errorf("出现错误！")
			}
			return messagemgr.ErrAutomaticHeavyThrow
		},
		Options: messagemgr.HandlerOptions{
			MaxRetry:    3,                               //重试3次
			AckWait:     time.Second * time.Duration(10), //最多等待10秒
			Concurrency: 100,                             //最多等待条未确认的消息
		},
	})
	err = mgr.Start()
	if err != nil {
		fmt.Println(err)
		return
	}
	go func() {
		for i := 0; i < 1; i++ {
			err := mgr.Publish(messagemgr.QueueTransport, "test", fmt.Sprintf("test-%d", i+1), "", 0)
			if err != nil {
				fmt.Println(err)
			}
			err = mgr.Publish(messagemgr.BroadcastTransport, "BroadcastTest", fmt.Sprintf("BroadcastTest-%d", i+1), "", 0)
			if err != nil {
				fmt.Println(err)
			}
		}
	}()

	select {}

}
