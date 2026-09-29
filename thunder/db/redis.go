// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 db
package db

// import 多行导入块开始
import (
	// 语句或表达式
	"context"
	// 语句或表达式
	"github.com/mszlu521/thunder/config"
	// 语句或表达式
	"github.com/redis/go-redis/v9"
	// 语句或表达式
	"time"
	// 闭合括号（import/var/参数列表等）
)

// 定义结构体类型 Redis
type Redis struct {
	// 语句或表达式
	Options *redis.Options
	// 语句或表达式
	Client *redis.Client
	// 语句或表达式
}

// 定义函数 Init（签名开始）
func (r *Redis) Init(redisConf *config.Redis) {
	// 条件判断 if
	if r.Options == nil {
		// 语句或表达式
		r.Options = &redis.Options{
			// 列表项（字段/参数/元素），行末逗号
			Addr: redisConf.GetAddr(),
			// 列表项（字段/参数/元素），行末逗号
			DB: redisConf.GetDB(),
			// 列表项（字段/参数/元素），行末逗号
			Password: redisConf.GetPassword(),
			// 列表项（字段/参数/元素），行末逗号
			PoolSize: redisConf.GetPoolSize(),
			// 列表项（字段/参数/元素），行末逗号
			MaxIdleConns: redisConf.GetMaxIdleConns(),
			// 列表项（字段/参数/元素），行末逗号
			MaxActiveConns: redisConf.GetMaxOpenConns(),
			// 语句或表达式
		}
		// 语句或表达式
	}
	// 短变量声明并赋值
	rdb := redis.NewClient(r.Options)
	// 短变量声明并赋值
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer 延迟执行
	defer cancel()
	// 短变量声明并赋值
	_, err := rdb.Ping(ctx).Result()
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		panic(err)
		// 语句或表达式
	}
	// 语句或表达式
	r.Client = rdb
	// 语句或表达式
}
