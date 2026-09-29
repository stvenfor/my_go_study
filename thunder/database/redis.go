// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 database
package database

// import 多行导入块开始
import (
	// 语句或表达式
	"github.com/mszlu521/thunder/config"
	// 语句或表达式
	"github.com/mszlu521/thunder/db"
	// 闭合括号（import/var/参数列表等）
)

// var 变量块开始
var (
	// 语句或表达式
	RedisCli *db.Redis

// 闭合括号（import/var/参数列表等）
)

// 定义函数 InitRedis（签名开始）
func InitRedis(redisConf *config.Redis) {
	// 条件判断 if
	if redisConf == nil {
		// 返回语句
		return
		// 语句或表达式
	}

	// 短变量声明并赋值
	r := db.Redis{}
	// 语句或表达式
	r.Init(redisConf)
	// 赋值语句
	RedisCli = &r
	// 语句或表达式
}
