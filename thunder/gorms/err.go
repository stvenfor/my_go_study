// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 gorms
package gorms

// import 多行导入块开始
import (
	// 语句或表达式
	"errors"

	// 语句或表达式
	"gorm.io/gorm"
	// 闭合括号（import/var/参数列表等）
)

// 定义函数 IsRecordNotFoundError（签名开始）
func IsRecordNotFoundError(err error) bool {
	// 返回语句
	return errors.Is(err, gorm.ErrRecordNotFound)
	// 语句或表达式
}
