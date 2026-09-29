// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 jwt
package jwt

// import 多行导入块开始
import (
	// 语句或表达式
	"errors"
	// 语句或表达式
	"time"

	// 语句或表达式
	"github.com/golang-jwt/jwt/v5"
	// 闭合括号（import/var/参数列表等）
)

// CustomClaims 自定义声明类型，内嵌 jwt.RegisteredClaims
// RegisteredClaims 包含官方定义的标准字段：(iss, sub, aud, exp, nbf, iat, jti)
// 我们这里增加了自定义字段 UserID 和 Username
// 定义结构体类型 CustomClaims
type CustomClaims struct {
	// 语句或表达式
	UserId string `json:"userId"`
	// 语句或表达式
	Username string `json:"username"`
	// 语句或表达式
	jwt.RegisteredClaims
	// 语句或表达式
}

// JWT 是我们的 JWT 工具结构体
// 定义结构体类型 JWT
type JWT struct {
	// SecretKey 用于签名的密钥，应该是保密的
	// 语句或表达式
	SecretKey []byte
	// 语句或表达式
}

// NewJWT 创建一个 JWT 工具实例
// 定义函数 NewJWT（签名开始）
func NewJWT(secretKey string) *JWT {
	// 返回语句
	return &JWT{
		// 列表项（字段/参数/元素），行末逗号
		SecretKey: []byte(secretKey),
		// 语句或表达式
	}
	// 语句或表达式
}

// GenerateToken 生成一个 token
// 它接收自定义的声明和过期时间
// 定义函数 GenerateToken（签名开始）
func (j *JWT) GenerateToken(claims CustomClaims, expirationTime time.Duration) (string, error) {
	// 设置标准声明的过期时间
	// jwt.NewNumericDate 是一个辅助函数，用于将 time.Time 转换为 JWT 使用的 Unix 时间戳
	// 语句或表达式
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(expirationTime))
	// 设置签发时间
	// 语句或表达式
	claims.RegisteredClaims.IssuedAt = jwt.NewNumericDate(time.Now())
	// 设置签发人 (可选)
	// claims.RegisteredClaims.Issuer = "my-project-name"

	// 使用指定的签名方法和声明创建一个新的 Token
	// 短变量声明并赋值
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用秘钥进行签名，并获取完整的编码后的字符串 token
	// 返回语句
	return token.SignedString(j.SecretKey)
	// 语句或表达式
}

// ParseToken 解析和验证一个 token
// 定义函数 ParseToken（签名开始）
func (j *JWT) ParseToken(tokenString string) (*CustomClaims, error) {
	// jwt.ParseWithClaims 是核心解析函数
	// 1. tokenString: 要解析的 token 字符串
	// 2. &CustomClaims{}: 一个空的声明对象指针，用于告诉库如何解码 payload
	// 3. Keyfunc: 一个回调函数，用于提供验证签名所需的密钥
	// 短变量声明并赋值
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 重要的安全校验：检查 token 使用的签名算法是否是我们期望的 HMAC 算法
		// 条件判断 if
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			// 返回语句
			return nil, errors.New("unexpected signing method")
			// 语句或表达式
		}
		// 返回语句
		return j.SecretKey, nil
		// 语句或表达式
	})

	// 条件判断 if
	if err != nil {
		// 这里处理各种解析错误
		// 条件判断 if
		if errors.Is(err, jwt.ErrTokenExpired) {
			// 返回语句
			return nil, errors.New("token has expired")
			// 语句或表达式
		}
		// 其他错误，例如签名无效、格式错误等
		// 返回语句
		return nil, errors.New("invalid token")
		// 语句或表达式
	}

	// 检查 token 是否有效，并且声明是否是我们定义的类型
	// 条件判断 if
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		// 返回语句
		return claims, nil
		// 语句或表达式
	}

	// 返回语句
	return nil, errors.New("invalid token")
	// 语句或表达式
}
