// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 req
package req

// import 多行导入块开始
import (
	// 语句或表达式
	"strconv"

	// 语句或表达式
	"github.com/gin-gonic/gin"
	// 语句或表达式
	"github.com/google/uuid"
	// 语句或表达式
	"github.com/mszlu521/thunder/errs"
	// 语句或表达式
	"github.com/mszlu521/thunder/logs"
	// 语句或表达式
	"github.com/mszlu521/thunder/res"
	// 闭合括号（import/var/参数列表等）
)

// 定义函数 JsonParam（签名开始）
func JsonParam(c *gin.Context, obj any) error {
	// 短变量声明并赋值
	err := c.ShouldBindJSON(obj)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		logs.Errorf("req parse json param err : %v", err)
		// 语句或表达式
		res.Error(c, errs.ErrParam)
		// 返回语句
		return errs.ErrParam
		// 语句或表达式
	}
	// 返回语句
	return nil
	// 语句或表达式
}

// 定义函数 QueryParam（签名开始）
func QueryParam(c *gin.Context, obj any) error {
	// 短变量声明并赋值
	err := c.ShouldBindQuery(obj)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		logs.Errorf("req parse query param err : %v", err)
		// 语句或表达式
		res.Error(c, errs.ErrParam)
		// 返回语句
		return errs.ErrParam
		// 语句或表达式
	}
	// 返回语句
	return nil
	// 语句或表达式
}

// 定义函数 XMLParam（签名开始）
func XMLParam(ctx *gin.Context, obj any) error {
	// 短变量声明并赋值
	err := ctx.ShouldBindBodyWithXML(obj)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		logs.Errorf("req parse xml param err : %v", err)
		// 返回语句
		return errs.ErrParam
		// 语句或表达式
	}
	// 返回语句
	return nil
	// 语句或表达式
}

// 定义函数 PathParam（签名开始）
func PathParam(ctx *gin.Context, paramKey string) string {
	// 短变量声明并赋值
	param := ctx.Param(paramKey)
	// 返回语句
	return param
	// 语句或表达式
}

// 定义函数 PathInt（签名开始）
func PathInt(ctx *gin.Context, paramKey string) (int64, error) {
	// 短变量声明并赋值
	param := PathParam(ctx, paramKey)
	// 条件判断 if
	if param == "" {
		// 语句或表达式
		res.Error(ctx, errs.ErrParam)
		// 返回语句
		return 0, errs.ErrParam
		// 语句或表达式
	}
	// 返回语句
	return StringToInt64(param)
	// 语句或表达式
}

// 定义函数 Path（签名开始）
func Path(ctx *gin.Context, paramKey string, value any) error {
	// 短变量声明并赋值
	param := PathParam(ctx, paramKey)
	// 条件判断 if
	if param == "" {
		// 语句或表达式
		logs.Errorf("req path parse param err : %v", "param is nil")
		// 语句或表达式
		res.Error(ctx, errs.ErrParam)
		// 返回语句
		return errs.ErrParam
		// 语句或表达式
	}

	// switch 分支
	switch v := value.(type) {
	// case 分支
	case *string:
		// 语句或表达式
		*v = param
	// case 分支
	case *int:
		// 短变量声明并赋值
		i, err := strconv.Atoi(param)
		// 条件判断 if
		if err != nil {
			// 语句或表达式
			logs.Errorf("req path parse param err : %v", err)
			// 语句或表达式
			res.Error(ctx, errs.ErrParam)
			// 返回语句
			return errs.ErrParam
			// 语句或表达式
		}
		// 语句或表达式
		*v = i
	// case 分支
	case *int64:
		// 短变量声明并赋值
		i, err := strconv.ParseInt(param, 10, 64)
		// 条件判断 if
		if err != nil {
			// 语句或表达式
			logs.Errorf("req path parse param err : %v", err)
			// 语句或表达式
			res.Error(ctx, errs.ErrParam)
			// 返回语句
			return errs.ErrParam
			// 语句或表达式
		}
		// 语句或表达式
		*v = i
	// case 分支
	case *uuid.UUID:
		// 变量声明
		var err error
		// 语句或表达式
		*v, err = uuid.Parse(param)
		// 条件判断 if
		if err != nil {
			// 语句或表达式
			logs.Errorf("req path parse param err : %v", err)
			// 语句或表达式
			res.Error(ctx, errs.ErrParam)
			// 返回语句
			return errs.ErrParam
			// 语句或表达式
		}
	// default 分支
	default:
		// 语句或表达式
		logs.Errorf("req path parse param err : %v", "no support param type")
		// 语句或表达式
		res.Error(ctx, errs.ErrParam)
		// 返回语句
		return errs.ErrParam
		// 语句或表达式
	}

	// 返回语句
	return nil
	// 语句或表达式
}

// 定义函数 StringToInt64（签名开始）
func StringToInt64(param string) (int64, error) {
	// 短变量声明并赋值
	i, err := strconv.ParseInt(param, 10, 64)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return 0, errs.ErrParam
		// 语句或表达式
	}
	// 返回语句
	return i, nil
	// 语句或表达式
}

// 定义函数 PathInArray（签名开始）
func PathInArray(ctx *gin.Context, method string, urls []string) bool {
	// 短变量声明并赋值
	path := ctx.Request.URL.Path
	// for 循环
	for _, url := range urls {
		// 条件判断 if
		if path == url && method == ctx.Request.Method {
			// 返回语句
			return true
			// 语句或表达式
		}
		// 语句或表达式
	}
	// 返回语句
	return false
	// 语句或表达式
}

// 定义函数 GetUserIdString（签名开始）
func GetUserIdString(ctx *gin.Context) (string, bool) {
	// 短变量声明并赋值
	value, exists := ctx.Get("userId")
	// 条件判断 if
	if !exists {
		// 语句或表达式
		res.Error(ctx, errs.ErrUnauthorized)
		// 返回语句
		return "", false
		// 语句或表达式
	}
	// 返回语句
	return value.(string), true
	// 语句或表达式
}

// 定义函数 GetUserId（签名开始）
func GetUserId(ctx *gin.Context) (int64, bool) {
	// 短变量声明并赋值
	value, exists := ctx.Get("userId")
	// 条件判断 if
	if !exists {
		// 语句或表达式
		res.Error(ctx, errs.ErrUnauthorized)
		// 返回语句
		return 0, false
		// 语句或表达式
	}
	// 返回语句
	return value.(int64), true
	// 语句或表达式
}

// 定义函数 GetUserIdUUID（签名开始）
func GetUserIdUUID(ctx *gin.Context) (uuid.UUID, bool) {
	// 短变量声明并赋值
	value, exists := ctx.Get("userId")
	// 条件判断 if
	if !exists {
		// 语句或表达式
		res.Error(ctx, errs.ErrUnauthorized)
		// 返回语句
		return uuid.Nil, false
		// 语句或表达式
	}
	// 短变量声明并赋值
	str := value.(string)
	// 短变量声明并赋值
	parse, err := uuid.Parse(str)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		res.Error(ctx, errs.ErrUnauthorized)
		// 返回语句
		return uuid.Nil, false
		// 语句或表达式
	}
	// 返回语句
	return parse, true
	// 语句或表达式
}
