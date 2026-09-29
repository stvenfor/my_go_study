// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 gptr
package gptr

// 语句或表达式
func Of[T any](v T) *T {
	// 返回语句
	return &v
	// 语句或表达式
}
