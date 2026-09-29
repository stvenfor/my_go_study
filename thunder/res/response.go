// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 res
package res

// import 多行导入块开始
import (
	// 语句或表达式
	"errors"
	// 语句或表达式
	"github.com/gin-gonic/gin"
	// 语句或表达式
	"github.com/mszlu521/thunder/errs"
	// 语句或表达式
	"net/http"
	// 闭合括号（import/var/参数列表等）
)

// 定义结构体类型 Result
type Result struct {
	// 语句或表达式
	Code int `json:"code"`
	// 语句或表达式
	Msg string `json:"msg"`
	// 语句或表达式
	Data any `json:"data"`
	// 语句或表达式
}

// 定义结构体类型 Page
type Page struct {
	// 语句或表达式
	Total int64 `json:"total"`
	// 语句或表达式
	List any `json:"list"`
	// 语句或表达式
	PageSize int64 `json:"pageSize"`
	// 语句或表达式
	CurrentPage int64 `json:"currentPage"`
	// 语句或表达式
}

// 定义函数 Success（签名开始）
func Success(ctx *gin.Context, data any) {
	// 语句或表达式
	ctx.JSON(http.StatusOK, Result{
		// 列表项（字段/参数/元素），行末逗号
		Code: OK,
		// 列表项（字段/参数/元素），行末逗号
		Data: data,
		// 语句或表达式
	})
	// 语句或表达式
}

// 定义函数 Error400（签名开始）
func Error400(ctx *gin.Context) {
	// 语句或表达式
	ctx.Writer.WriteHeader(http.StatusBadRequest)
	// 语句或表达式
}

// 定义函数 Error500（签名开始）
func Error500(ctx *gin.Context) {
	// 语句或表达式
	ctx.Writer.WriteHeader(http.StatusInternalServerError)
	// 语句或表达式
}

// 定义函数 Error（签名开始）
func Error(ctx *gin.Context, err error) {
	// 变量声明
	var er *errs.Errors
	// switch 分支
	switch {
	// case 分支
	case errors.As(err, &er):
		// 条件判断 if
		if errors.Is(er, errs.ErrParam) {
			// 语句或表达式
			ctx.Writer.WriteHeader(http.StatusBadRequest)
			// 语句或表达式
		} else if errors.Is(er, errs.ErrUnauthorized) {
			// 语句或表达式
			ctx.Writer.WriteHeader(http.StatusUnauthorized)
			// 语句或表达式
		} else {
			// 语句或表达式
			Fail(ctx, er)
			// 语句或表达式
		}
	// default 分支
	default:
		// 语句或表达式
		Error500(ctx)
		// 语句或表达式
	}
	// 语句或表达式
}

// 定义函数 Fail（签名开始）
func Fail(ctx *gin.Context, e *errs.Errors) {
	// 语句或表达式
	ctx.JSON(http.StatusOK, Result{
		// 列表项（字段/参数/元素），行末逗号
		Code: e.Code,
		// 列表项（字段/参数/元素），行末逗号
		Msg: e.Msg,
		// 语句或表达式
	})
	// 语句或表达式
}

// 定义函数 SetCookie（签名开始）
func SetCookie(ctx *gin.Context, key string, value string, expire int64) {
	// 语句或表达式
	http.SetCookie(ctx.Writer, &http.Cookie{
		// 列表项（字段/参数/元素），行末逗号
		Name: key,
		// 列表项（字段/参数/元素），行末逗号
		Value: value,
		// 列表项（字段/参数/元素），行末逗号
		MaxAge: int(expire),
		// 列表项（字段/参数/元素），行末逗号
		Path: "/",
		// 语句或表达式
	})
	// 语句或表达式
}
