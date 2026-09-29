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
	_db *db.MySQL

// 闭合括号（import/var/参数列表等）
)

// 定义函数 InitDB（签名开始）
func InitDB(mysqlConf *config.Mysql) {
	// 条件判断 if
	if mysqlConf == nil {
		// 返回语句
		return
		// 语句或表达式
	}

	// 短变量声明并赋值
	m := db.MySQL{
		// 列表项（字段/参数/元素），行末逗号
		Database: mysqlConf.GetDatabase(),
		// 列表项（字段/参数/元素），行末逗号
		Host: mysqlConf.GetHost(),
		// 列表项（字段/参数/元素），行末逗号
		MaxIdleConns: mysqlConf.GetMaxIdleConns(),
		// 列表项（字段/参数/元素），行末逗号
		MaxOpenConns: mysqlConf.GetMaxOpenConns(),
		// 列表项（字段/参数/元素），行末逗号
		Password: mysqlConf.GetPassword(),
		// 列表项（字段/参数/元素），行末逗号
		Port: mysqlConf.GetPort(),
		// 列表项（字段/参数/元素），行末逗号
		Username: mysqlConf.GetUser(),
		// 列表项（字段/参数/元素），行末逗号
		PingTimeout: mysqlConf.GetPingTimeout(),
		// 语句或表达式
	}
	// 短变量声明并赋值
	err := m.Init()
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		panic(err)
		// 语句或表达式
	}
	// 赋值语句
	_db = &m
	// 语句或表达式
}

// 定义函数 GetMysqlDB（签名开始）
func GetMysqlDB() *db.MySQL {
	// 返回语句
	return _db
	// 语句或表达式
}
