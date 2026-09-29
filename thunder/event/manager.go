// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 event
package event

// 导入依赖包 "github.com/mszlu521/thunder/errs"
import "github.com/mszlu521/thunder/errs"

// 定义结构体类型 Event
type Event struct {
	// 语句或表达式
	Name string
	// 语句或表达式
	Data any
	// 语句或表达式
}

// 类型定义
type Handler func(e Event) (any, error)

// 变量声明
var handlers = make(map[string]Handler)

// 定义函数 Register（签名开始）
func Register(eventName string, handler Handler) {
	// 语句或表达式
	handlers[eventName] = handler
	// 语句或表达式
}

// 定义函数 Trigger（签名开始）
func Trigger(eventName string, data interface{}) (any, error) {
	// 条件判断 if
	if handler, exists := handlers[eventName]; exists {
		// 返回语句
		return handler(Event{Name: eventName, Data: data})
		// 语句或表达式
	}
	// 返回语句
	return nil, errs.NoEventHandler
	// 语句或表达式
}
