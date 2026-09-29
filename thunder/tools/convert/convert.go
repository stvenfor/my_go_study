// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 convert
package convert

// 导入依赖包 "strconv"
import "strconv"

// 定义函数 GetInterfaceToInt（签名开始）
func GetInterfaceToInt(t1 interface{}) int64 {
	// 变量声明
	var t2 int64
	// switch 分支
	switch t1.(type) {
	// case 分支
	case uint:
		// 赋值语句
		t2 = int64(t1.(uint))
		// 语句或表达式
		break
	// case 分支
	case int8:
		// 赋值语句
		t2 = int64(t1.(int8))
		// 语句或表达式
		break
	// case 分支
	case uint8:
		// 赋值语句
		t2 = int64(t1.(uint8))
		// 语句或表达式
		break
	// case 分支
	case int16:
		// 赋值语句
		t2 = int64(t1.(int16))
		// 语句或表达式
		break
	// case 分支
	case uint16:
		// 赋值语句
		t2 = int64(t1.(uint16))
		// 语句或表达式
		break
	// case 分支
	case int32:
		// 赋值语句
		t2 = int64(t1.(int32))
		// 语句或表达式
		break
	// case 分支
	case uint32:
		// 赋值语句
		t2 = int64(t1.(uint32))
		// 语句或表达式
		break
	// case 分支
	case int64:
		// 赋值语句
		t2 = int64(t1.(int64))
		// 语句或表达式
		break
	// case 分支
	case uint64:
		// 赋值语句
		t2 = int64(t1.(uint64))
		// 语句或表达式
		break
	// case 分支
	case float32:
		// 赋值语句
		t2 = int64(t1.(float32))
		// 语句或表达式
		break
	// case 分支
	case float64:
		// 赋值语句
		t2 = int64(t1.(float64))
		// 语句或表达式
		break
	// case 分支
	case string:
		// 语句或表达式
		t2, _ = strconv.ParseInt(t1.(string), 10, 64)
		// 语句或表达式
		break
	// default 分支
	default:
		// 赋值语句
		t2 = t1.(int64)
		// 语句或表达式
		break
		// 语句或表达式
	}
	// 返回语句
	return t2
	// 语句或表达式
}
