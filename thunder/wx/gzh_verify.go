// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 wx
package wx

// import 多行导入块开始
import (
	// 语句或表达式
	"github.com/gin-gonic/gin"
	// 语句或表达式
	"github.com/mszlu521/thunder/req"
	// 语句或表达式
	"github.com/mszlu521/thunder/res"
	// 语句或表达式
	"github.com/mszlu521/thunder/tools/crypro"
	// 语句或表达式
	"sort"
	// 语句或表达式
	"strings"
	// 闭合括号（import/var/参数列表等）
)

// 定义结构体类型 TokenVerifyReq
type TokenVerifyReq struct {
	// 语句或表达式
	Signature string `json:"signature" form:"signature"`
	// 语句或表达式
	Timestamp string `json:"timestamp" form:"timestamp"`
	// 语句或表达式
	EchoStr string `json:"echostr" form:"echostr"`
	// 语句或表达式
	Nonce string `json:"nonce" form:"nonce"`
	// 语句或表达式
}

// 定义函数 verify（签名开始）
func (r *TokenVerifyReq) verify(token string) bool {
	// 短变量声明并赋值
	arr := []string{token, r.Timestamp, r.Nonce}
	// 语句或表达式
	sort.Slice(arr, func(i, j int) bool {
		// 返回语句
		return strings.ToLower(arr[i]) < strings.ToLower(arr[j])
		// 语句或表达式
	})
	// 短变量声明并赋值
	sign := strings.Join(arr, "")
	// 赋值语句
	sign = crypro.Sha1(sign)
	// 条件判断 if
	if sign == r.Signature {
		// 返回语句
		return true
		// 语句或表达式
	}
	// 返回语句
	return false
	// 语句或表达式
}

// 定义函数 Verify（签名开始）
func Verify(token string, method string, urls []string) gin.HandlerFunc {
	// 返回语句
	return func(ctx *gin.Context) {
		// 条件判断 if
		if !req.PathInArray(ctx, method, urls) {
			// 语句或表达式
			ctx.Abort()
			// 返回语句
			return
			// 语句或表达式
		}
		// 变量声明
		var tokenVerify TokenVerifyReq
		// 条件判断 if
		if err := req.QueryParam(ctx, &tokenVerify); err != nil {
			// 语句或表达式
			res.Error(ctx, err)
			// 返回语句
			return
			// 语句或表达式
		}
		// 条件判断 if
		if !tokenVerify.verify(token) {
			// 语句或表达式
			ctx.Abort()
			// 语句或表达式
		}
		// 语句或表达式
		ctx.Next()
		// 语句或表达式
	}
	// 语句或表达式
}
