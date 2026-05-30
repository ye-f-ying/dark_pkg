/*
 * @Date: 2026-05-30 16:10:27
 * @LastEditTime: 2026-05-30 16:59:27
 * @FilePath: /dark_pkg/pkg/config/redis_config.go
 * @Description:
 */
package config

type REDIS struct {
	Address      []string `mapstructure:"address"`
	UserName     string   `mapstructure:"user_name"`
	Password     string   `mapstructure:"password"`                      // 密码
	DB           int      `mapstructure:"db" `                           // db
	PoolSize     int      `mapstructure:"pool_size"`                     // 连接池大小
	MinIdleConns int      `mapstructure:"min_idle_conns" default:"1"`    // 最小空闲连接
	ReadTimeout  int      `mapstructure:"read_timeout" default:"3000"`   // 读取超时 毫秒
	WriteTimeout int      `mapstructure:"write_timeout"  default:"3000"` // 写入超时 毫秒
}
