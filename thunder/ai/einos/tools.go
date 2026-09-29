// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 einos
package einos

// import 多行导入块开始
import (
	// 语句或表达式
	"github.com/cloudwego/eino/components/tool"
	// 语句或表达式
	"github.com/cloudwego/eino/schema"
	// 闭合括号（import/var/参数列表等）
)

// 定义接口类型 InvokeParamTool
type InvokeParamTool interface {
	// 语句或表达式
	tool.InvokableTool

	Params() map[string]*schema.ParameterInfo
}
