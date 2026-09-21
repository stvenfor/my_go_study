// user_usecase.go 存放认证入参与业务错误。本地账号读写在 LocalAuthUsecase。
package usecase

import "errors"

var (
	// ErrInvalidParams 参数不合法。
	ErrInvalidParams = errors.New("invalid params")
	// ErrUserExists 用户已存在。
	ErrUserExists = errors.New("user already exists")
	// ErrInvalidCredentials 用户名或密码错误。
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserNotFound 用户不存在。
	ErrUserNotFound = errors.New("user not found")
)

// RegisterInput 注册入参。客户端传入的 user_id 不在此结构中，注册时忽略。
type RegisterInput struct {
	Username string
	Password string
	Email    string
}

// LoginInput 登录入参。
type LoginInput struct {
	Username string
	Password string
}
