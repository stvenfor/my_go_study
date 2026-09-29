// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 timeouts
package timeouts

// import 多行导入块开始
import (
	// 语句或表达式
	"context"
	// 语句或表达式
	"time"
	// 闭合括号（import/var/参数列表等）
)

// 定义结构体类型 Executor
type Executor struct {
	// 语句或表达式
	timeout time.Duration
	// 语句或表达式
}

// 定义函数 Execute（签名开始）
func (e *Executor) Execute(fn func(ctx context.Context) error) error {
	// 短变量声明并赋值
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	// defer 延迟执行
	defer cancel()
	// 返回语句
	return fn(ctx)
	// 语句或表达式
}
