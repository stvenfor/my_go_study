// user_request.go 定义用户相关 HTTP 请求 DTO。
package request

// LoginRequest 用户登录请求体。
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	DeviceID string `json:"device_id" binding:"required"`
	Platform string `json:"platform" binding:"required,oneof=android ios"`
}

// RegisterRequest 用户注册请求体。
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=64"`
	Email    string `json:"email" binding:"required,email,max=128"`
	DeviceID string `json:"device_id" binding:"required"`
	Platform string `json:"platform" binding:"required,oneof=android ios"`
}

// SendPhoneOTPRequest 发送手机验证码请求体。
type SendPhoneOTPRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// VerifyPhoneOTPRequest 校验手机验证码并登录请求体。
type VerifyPhoneOTPRequest struct {
	Phone    string `json:"phone" binding:"required"`
	OTP      string `json:"otp" binding:"required"`
	DeviceID string `json:"device_id" binding:"required"`
	Platform string `json:"platform" binding:"required,oneof=android ios"`
}
