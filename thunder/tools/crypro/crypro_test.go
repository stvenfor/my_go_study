// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 crypro
package crypro

// import 多行导入块开始
import (
	// 语句或表达式
	"fmt"
	// 语句或表达式
	"testing"
	// 闭合括号（import/var/参数列表等）
)

// 定义函数 TestEncryptString（签名开始）
func TestEncryptString(t *testing.T) {
	// 语句或表达式
	fmt.Println(Md5WithSalt("ms-co!@#$12398mmx", "ZfnXw"))
	// 语句或表达式
}
