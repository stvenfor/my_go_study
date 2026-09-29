// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 helper
package helper

// 定义函数 StringPtr（签名开始）
func StringPtr(s string) *string {
	// 返回语句
	return &s
	// 语句或表达式
}

// 定义函数 BoolPtr（签名开始）
func BoolPtr(b bool) *bool {
	// 返回语句
	return &b
	// 语句或表达式
}
