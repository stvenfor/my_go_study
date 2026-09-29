// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 errs
package errs

// 变量声明
var ErrParam = NewError(400, "param error")

// 变量声明
var ErrUnauthorized = NewError(401, "unauthorized")

// 变量声明
var NoEventHandler = NewError(500, "no handler")

// 变量声明
var DBError = NewError(999, "db error")

// 定义结构体类型 Errors
type Errors struct {
	// 语句或表达式
	Code int `json:"code"`
	// 语句或表达式
	Msg string `json:"msg"`
	// 语句或表达式
}

// 定义函数 Error（签名开始）
func (e *Errors) Error() string {
	// 返回语句
	return e.Msg
	// 语句或表达式
}

// 定义函数 NewError（签名开始）
func NewError(code int, msg string) *Errors {
	// 返回语句
	return &Errors{
		// 列表项（字段/参数/元素），行末逗号
		Code: code,
		// 列表项（字段/参数/元素），行末逗号
		Msg: msg,
		// 语句或表达式
	}
	// 语句或表达式
}
