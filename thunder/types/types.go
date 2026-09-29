// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 types
package types

// import 多行导入块开始
import (
	// 语句或表达式
	"database/sql/driver"
	// 语句或表达式
	"encoding/json"
	// 语句或表达式
	"errors"
	// 语句或表达式
	"strconv"
	// 语句或表达式
	"strings"
	// 闭合括号（import/var/参数列表等）
)

// 类型定义
type ArrayString []string

// 定义函数 Scan（签名开始）
func (a *ArrayString) Scan(value interface{}) error {
	// 短变量声明并赋值
	bytes, ok := value.([]byte)
	// 条件判断 if
	if !ok {
		// 返回语句
		return errors.New("type assertion to []byte failed")
		// 语句或表达式
	}
	// 变量声明
	var as ArrayString
	// 短变量声明并赋值
	err := json.Unmarshal(bytes, &as)
	// 语句或表达式
	*a = as
	// 返回语句
	return err
	// 语句或表达式
}

// 定义函数 Value（签名开始）
func (a ArrayString) Value() (driver.Value, error) {
	// 条件判断 if
	if a == nil || len(a) == 0 {
		// 返回语句
		return nil, nil
		// 语句或表达式
	}
	// 返回语句
	return json.Marshal(a)
	// 语句或表达式
}

// 定义函数 GormDataType（签名开始）
func (*ArrayString) GormDataType() string {
	// 返回语句
	return "varchar"
	// 语句或表达式
}

// 类型定义
type Int64String int64

// MarshalJSON 实现 MarshalJSON 接口
// 定义函数 MarshalJSON（签名开始）
func (i Int64String) MarshalJSON() ([]byte, error) {
	// 返回语句
	return []byte(`"` + strconv.FormatInt(int64(i), 10) + `"`), nil
	// 语句或表达式
}

// UnmarshalJSON 实现 UnmarshalJSON 接口
// 定义函数 UnmarshalJSON（签名开始）
func (i *Int64String) UnmarshalJSON(data []byte) error {
	// 短变量声明并赋值
	strData := string(data)
	// 条件判断 if
	if strings.Contains(strData, "\"") {
		// 赋值语句
		strData = strings.Replace(strData, "\"", "", -1)
		// 语句或表达式
	}
	// 短变量声明并赋值
	parsedInt, err := strconv.ParseInt(strData, 10, 64)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return err
		// 语句或表达式
	}
	// 语句或表达式
	*i = Int64String(parsedInt)
	// 返回语句
	return nil
	// 语句或表达式
}
