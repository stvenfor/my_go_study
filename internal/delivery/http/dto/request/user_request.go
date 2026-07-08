// user_request.go 定义用户相关 HTTP 请求 DTO。
package request

// RegisterRequest 用户注册请求体。
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=64"`
	Email    string `json:"email" binding:"required,email,max=128"`
}

// LoginRequest 用户登录请求体。
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
