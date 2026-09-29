// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 req
package req

// 导入依赖包 "github.com/gin-gonic/gin"
import "github.com/gin-gonic/gin"

// 定义函数 GetInt64（签名开始）
func GetInt64(c *gin.Context, key string) int64 {
	// 短变量声明并赋值
	value, ok := c.Get(key)
	// 条件判断 if
	if ok {
		// 返回语句
		return int64(value.(float64))
		// 语句或表达式
	}
	// 返回语句
	return 0
	// 语句或表达式
}
