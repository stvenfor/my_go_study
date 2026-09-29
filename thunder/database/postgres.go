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
	_pg *db.Postgres

// 闭合括号（import/var/参数列表等）
)

// 定义函数 InitPostgres（签名开始）
func InitPostgres(pgConf *config.Postgres) {
	// 条件判断 if
	if pgConf == nil {
		// 返回语句
		return
		// 语句或表达式
	}

	// 短变量声明并赋值
	p := db.Postgres{
		// 列表项（字段/参数/元素），行末逗号
		Database: pgConf.GetDatabase(),
		// 列表项（字段/参数/元素），行末逗号
		Host: pgConf.GetHost(),
		// 列表项（字段/参数/元素），行末逗号
		MaxIdleConns: pgConf.GetMaxIdleConns(),
		// 列表项（字段/参数/元素），行末逗号
		MaxOpenConns: pgConf.GetMaxOpenConns(),
		// 列表项（字段/参数/元素），行末逗号
		Password: pgConf.GetPassword(),
		// 列表项（字段/参数/元素），行末逗号
		Port: pgConf.GetPort(),
		// 列表项（字段/参数/元素），行末逗号
		Username: pgConf.GetUser(),
		// 列表项（字段/参数/元素），行末逗号
		SSLMode: pgConf.GetSSLMode(),
		// 列表项（字段/参数/元素），行末逗号
		PingTimeout: pgConf.GetPingTimeout(),
		// 语句或表达式
	}
	// 短变量声明并赋值
	err := p.Init()
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		panic(err)
		// 语句或表达式
	}
	// 赋值语句
	_pg = &p
	// 语句或表达式
}

// 定义函数 GetPostgresDB（签名开始）
func GetPostgresDB() *db.Postgres {
	// 返回语句
	return _pg
	// 语句或表达式
}
