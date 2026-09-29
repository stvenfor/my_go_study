// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 jwt
package jwt

// 导入依赖包 "time"
import "time"

// 变量声明
var _jwt *JWT

// Init 使用之前必须先调用Init初始化
// 定义函数 Init（签名开始）
func Init(secretKey string) {
	// 赋值语句
	_jwt = NewJWT(secretKey)
	// 语句或表达式
}

// 定义函数 GenerateToken（签名开始）
func GenerateToken(claims CustomClaims, expirationTime time.Duration) (string, error) {
	// 返回语句
	return _jwt.GenerateToken(claims, expirationTime)
	// 语句或表达式
}

// 定义函数 GenToken（签名开始）
func GenToken(userId string, username string, expirationTime time.Duration) (string, error) {
	// 短变量声明并赋值
	claims := CustomClaims{
		// 列表项（字段/参数/元素），行末逗号
		UserId: userId,
		// 列表项（字段/参数/元素），行末逗号
		Username: username,
		// 语句或表达式
	}
	// 返回语句
	return _jwt.GenerateToken(claims, expirationTime)
	// 语句或表达式
}

// 定义函数 ParseToken（签名开始）
func ParseToken(tokenString string) (*CustomClaims, error) {
	// 返回语句
	return _jwt.ParseToken(tokenString)
	// 语句或表达式
}
