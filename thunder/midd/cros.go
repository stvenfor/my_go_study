// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 midd
package midd

// import 多行导入块开始
import (
	// 语句或表达式
	"net/http"
	// 语句或表达式
	"strings"

	// 语句或表达式
	"github.com/gin-gonic/gin"
	// 语句或表达式
	"github.com/mszlu521/thunder/config"
	// 闭合括号（import/var/参数列表等）
)

// 定义函数 Cors（签名开始）
func Cors(conf *config.Server) gin.HandlerFunc {
	// 返回语句
	return func(c *gin.Context) {
		// 短变量声明并赋值
		origin := c.Request.Header.Get("Origin")
		// 短变量声明并赋值
		allowedOrigin, ok := isOriginAllowed(origin, conf.Cros)

		// 条件判断 if
		if ok {
			// 语句或表达式
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			// 当允许的源不是 '*' 时，才设置 credentials
			// 条件判断 if
			if allowedOrigin != "*" {
				// 语句或表达式
				c.Header("Access-Control-Allow-Credentials", "true")
				// 语句或表达式
			}
			// 语句或表达式
		}
		// 短变量声明并赋值
		method := c.Request.Method
		// 语句或表达式
		c.Header("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, OPTIONS")
		// 语句或表达式
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, Content-SessionType, Token")
		// 语句或表达式
		c.Header("Access-Control-Expose-Headers", "Access-Control-Allow-Headers, Token")
		// 语句或表达式
		c.Header("Access-Control-Max-Age", "172800")
		// 语句或表达式
		c.Header("Access-Control-Allow-Credentials", "true")
		// 条件判断 if
		if method == "OPTIONS" {
			//c.JSON(200, Controller.R(200, nil, "Options Request"))
			// 语句或表达式
			c.AbortWithStatus(http.StatusNoContent)
			// 语句或表达式
		}
		// 语句或表达式
		c.Next()
		// 语句或表达式
	}
	// 语句或表达式
}

// 定义函数 isOriginAllowed（签名开始）
func isOriginAllowed(origin string, allowedOrigins []string) (string, bool) {
	// 完全匹配 "*"
	// for 循环
	for _, o := range allowedOrigins {
		// 条件判断 if
		if o == "*" {
			// 返回语句
			return "*", true
			// 语句或表达式
		}
		// 语句或表达式
	}
	// 完全匹配域名
	// for 循环
	for _, o := range allowedOrigins {
		// 条件判断 if
		if o == origin {
			// 返回语句
			return origin, true
			// 语句或表达式
		}
		// 语句或表达式
	}
	// 匹配子域通配符, e.g., *.example.com
	// for 循环
	for _, o := range allowedOrigins {
		// 条件判断 if
		if strings.HasPrefix(o, "*.") {
			// 短变量声明并赋值
			domainSuffix := strings.TrimPrefix(o, "*.")
			// 条件判断 if
			if strings.HasSuffix(origin, "."+domainSuffix) {
				// 找到了匹配的子域，返回具体的 origin
				// 返回语句
				return origin, true
				// 语句或表达式
			}
			// 语句或表达式
		}
		// 语句或表达式
	}
	// 返回语句
	return "", false
	// 语句或表达式
}
