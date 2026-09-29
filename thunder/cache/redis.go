// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 cache
package cache

// import 多行导入块开始
import (
	// 语句或表达式
	"context"
	// 语句或表达式
	"errors"
	// 语句或表达式
	"github.com/mszlu521/thunder/database"
	// 语句或表达式
	"github.com/redis/go-redis/v9"
	// 语句或表达式
	"time"
	// 闭合括号（import/var/参数列表等）
)

// 定义结构体类型 RedisCache
type RedisCache struct {
	// 语句或表达式
	redisCli *redis.Client
	// 语句或表达式
}

// 定义函数 Get（签名开始）
func (c *RedisCache) Get(key string) (string, error) {
	// 短变量声明并赋值
	result, err := c.redisCli.Get(context.Background(), key).Result()
	// 条件判断 if
	if err != nil && errors.Is(err, redis.Nil) {
		// 返回语句
		return "", nil
		// 语句或表达式
	}
	// 返回语句
	return result, err
	// 语句或表达式
}

// 定义函数 TTL（签名开始）
func (c *RedisCache) TTL(key string) (time.Duration, error) {
	// 短变量声明并赋值
	result, err := c.redisCli.TTL(context.Background(), key).Result()
	// 条件判断 if
	if err != nil {
		// 条件判断 if
		if errors.Is(err, redis.Nil) {
			// 返回语句
			return 0, nil
			// 语句或表达式
		}
		// 语句或表达式
	}
	// 返回语句
	return result, err
	// 语句或表达式
}

// 定义函数 GetValueAndTTL（签名开始）
func (c *RedisCache) GetValueAndTTL(ctx context.Context, key string) (string, time.Duration, error) {
	// 短变量声明并赋值
	rdb := c.redisCli
	// 短变量声明并赋值
	pipe := rdb.Pipeline()
	// 短变量声明并赋值
	getCmd := pipe.Get(ctx, key)
	// 短变量声明并赋值
	ttlCmd := pipe.TTL(ctx, key)
	// 短变量声明并赋值
	_, err := pipe.Exec(ctx)
	// 条件判断 if
	if err != nil {
		// 条件判断 if
		if errors.Is(err, redis.Nil) {
			// 返回语句
			return "", 0, nil
			// 语句或表达式
		}
		// 返回语句
		return "", 0, err
		// 语句或表达式
	}
	// 短变量声明并赋值
	val, err := getCmd.Result()
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", 0, err
		// 语句或表达式
	}

	// 短变量声明并赋值
	ttl, err := ttlCmd.Result()
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", 0, err
		// 语句或表达式
	}
	// 返回语句
	return val, ttl, nil
	// 语句或表达式
}

// 定义函数 Set（签名开始）
func (c *RedisCache) Set(key string, value string, expire int64) error {
	// 返回语句
	return c.redisCli.Set(context.Background(), key, value, time.Duration(expire)*time.Second).Err()
	// 语句或表达式
}

// 定义函数 Exist（签名开始）
func (c *RedisCache) Exist(key string) bool {
	// 短变量声明并赋值
	result, err := c.redisCli.Exists(context.Background(), key).Result()
	// 返回语句
	return result == 1 && err == nil
	// 语句或表达式
}

// 定义函数 NewRedisCache（签名开始）
func NewRedisCache() *RedisCache {
	// 返回语句
	return &RedisCache{
		// 列表项（字段/参数/元素），行末逗号
		redisCli: database.RedisCli.Client,
		// 语句或表达式
	}
	// 语句或表达式
}
