// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 randoms
package randoms

// import 多行导入块开始
import (
	// 语句或表达式
	cr "crypto/rand"
	// 语句或表达式
	"fmt"
	// 语句或表达式
	"math/rand"
	// 语句或表达式
	v2 "math/rand/v2"
	// 语句或表达式
	"time"
	// 闭合括号（import/var/参数列表等）
)

// 定义函数 Generate4Number（签名开始）
func Generate4Number() int {
	// 使用当前时间作为种子
	// 短变量声明并赋值
	source := rand.NewSource(time.Now().UnixNano())
	// 短变量声明并赋值
	r := rand.New(source)
	// 生成一个范围在 1000 到 9999 之间的随机数
	// 短变量声明并赋值
	randomNumber := r.Intn(9000) + 1000
	// 返回语句
	return randomNumber
	// 语句或表达式
}

// 定义函数 Generate6Number（签名开始）
func Generate6Number() int {
	// 使用当前时间作为种子
	// 短变量声明并赋值
	source := rand.NewSource(time.Now().UnixNano())
	// 短变量声明并赋值
	r := rand.New(source)
	// 短变量声明并赋值
	randomNumber := r.Intn(900000) + 100000
	// 返回语句
	return randomNumber
	// 语句或表达式
}

// 定义函数 GenerateTicket（签名开始）
func GenerateTicket() string {
	// 短变量声明并赋值
	timestamp := time.Now().UnixNano() // 获取当前时间的纳秒级时间戳
	// 短变量声明并赋值
	randomBytes := make([]byte, 16) // 生成16字节随机数
	// 语句或表达式
	_, _ = cr.Read(randomBytes)
	// 返回语句
	return fmt.Sprintf("%x%x", timestamp, randomBytes) // 拼接时间戳和随机数
	// 语句或表达式
}

// Gen6Code 生成6位数字验证码
// 定义函数 Gen6Code（签名开始）
func Gen6Code() (string, error) {
	// 1. 准备种子
	// 变量声明
	var seed [32]byte
	// 条件判断 if
	if _, err := cr.Read(seed[:]); err != nil {
		// 返回语句
		return "", err
		// 语句或表达式
	}
	// 2. 创建随机源
	// 短变量声明并赋值
	cha8 := v2.NewChaCha8(seed)
	// 3. 基于源创建一个随机生成器对象
	// 短变量声明并赋值
	r := v2.New(cha8)
	// 4. 直接请求一个 [0, 1000000) 的随机整数
	// IntN 会自动处理边界，比自己做取模运算更均匀（没有模偏差）
	// 短变量声明并赋值
	num := r.IntN(1000000)
	// 5. 格式化为 6 位字符串 (不足补0)
	// 返回语句
	return fmt.Sprintf("%06d", num), nil
	// 语句或表达式
}

// Gen6CodeNumber 生成6位数字验证码
// 定义函数 Gen6CodeNumber（签名开始）
func Gen6CodeNumber() (int, error) {
	// 1. 准备种子
	// 变量声明
	var seed [32]byte
	// 条件判断 if
	if _, err := cr.Read(seed[:]); err != nil {
		// 返回语句
		return 0, err
		// 语句或表达式
	}
	// 2. 创建随机源
	// 短变量声明并赋值
	cha8 := v2.NewChaCha8(seed)
	// 3. 基于源创建一个随机生成器对象
	// 短变量声明并赋值
	r := v2.New(cha8)
	// 4. 直接请求一个 [0, 1000000) 的随机整数
	// IntN 会自动处理边界，比自己做取模运算更均匀（没有模偏差）
	// 短变量声明并赋值
	num := r.IntN(1000000)
	// 5. 格式化为 6 位字符串 (不足补0)
	// 返回语句
	return num, nil
	// 语句或表达式
}
