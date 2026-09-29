// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 server
package server

// import 多行导入块开始
import (
	// 语句或表达式
	"github.com/gin-gonic/gin"
	// 语句或表达式
	"github.com/mszlu521/thunder/config"
	// 语句或表达式
	"github.com/mszlu521/thunder/midd"
	// 闭合括号（import/var/参数列表等）
)

// IRouter 定义路由注册接口
// 定义接口类型 IRouter
type IRouter interface {
	// 语句或表达式
	Register(engine *gin.Engine)
	// 语句或表达式
}

// 定义接口类型 CloseIRouter
type CloseIRouter interface {
	// 语句或表达式
	IRouter
	// 语句或表达式
	Close() error
	// 语句或表达式
}

// 定义函数 UseCustomMidd（签名开始）
func UseCustomMidd(conf *config.Config, engin *gin.Engine) {
	// 条件判断 if
	if conf.Server != nil {
		// 条件判断 if
		if len(conf.Server.GetCros()) > 0 {
			// 语句或表达式
			engin.Use(midd.Cors(conf.Server))
			// 语句或表达式
		}
		// 语句或表达式
	}
	// 条件判断 if
	if conf.Auth != nil {
		// 条件判断 if
		if conf.Auth.GetIsAuth() {
			// 语句或表达式
			engin.Use(midd.Auth(conf.Auth))
			// 语句或表达式
		}
		// 语句或表达式
	}
	// 条件判断 if
	if conf.Cache != nil {
		// 条件判断 if
		if len(conf.Cache.GetNeedCache()) > 0 {
			// 语句或表达式
			engin.Use(midd.Cache(conf.Cache))
			// 语句或表达式
		}
		// 语句或表达式
	}
	// 语句或表达式
}
