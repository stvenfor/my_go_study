// Package authctx 会话身份在 gin.Context 上的读写助手（叶子包）。
//
// 为什么单独成包：middleware 必须依赖 usecase（LocalSessionAuth /
// SupabaseSessionAuth 要调 *usecase.DeviceSessionUsecase），而各域
// （access / home / points / wallet …）的 handler 也要取身份。若域直接
// import middleware，就会形成
//
//	usecase → 域(access) → middleware → usecase
//
// 的导入环。本包只依赖 gin + common/auth，域与 middleware 都能安全依赖。
// middleware 里同名函数保留为转发层，供尚未切片的 controller 使用。
package authctx

import (
	"strings"

	pkgauth "common/auth"
	"github.com/gin-gonic/gin"
)

const (
	ContextSupabaseUserKey = "supabaseUser" // c.Get 取用户
	ContextAccessTokenKey  = "accessToken"  // 转发 PostgREST 时需要原 token
)

// SetIdentity 写入会话身份（由 LocalSessionAuth / SupabaseSessionAuth 调用）。
func SetIdentity(c *gin.Context, user pkgauth.SupabaseUser, token string) {
	c.Set(ContextSupabaseUserKey, user)
	c.Set(ContextAccessTokenKey, token)
}

// GetSupabaseUser 取会话用户。
func GetSupabaseUser(c *gin.Context) (pkgauth.SupabaseUser, bool) {
	value, ok := c.Get(ContextSupabaseUserKey)
	if !ok {
		return pkgauth.SupabaseUser{}, false
	}
	user, ok := value.(pkgauth.SupabaseUser)
	return user, ok
}

// GetAccessToken 取原始访问令牌。
func GetAccessToken(c *gin.Context) (string, bool) {
	value, ok := c.Get(ContextAccessTokenKey)
	if !ok {
		return "", false
	}
	token, ok := value.(string)
	return token, ok
}

// BearerToken 解析 "Bearer eyJ..." → "eyJ..."；无前缀时原样返回（去空白）。
func BearerToken(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(header, prefix))
	}
	return strings.TrimSpace(header)
}

// SupabaseAuthContext 读取会话用户与访问令牌；缺任一即视为未授权。
//
// 这是各域 handler 的统一取身份入口（本地 provider 模式下 LocalSessionAuth
// 写入的是同一组上下文键，两边共用本函数）。
func SupabaseAuthContext(c *gin.Context) (pkgauth.SupabaseUser, string, bool) {
	user, ok := GetSupabaseUser(c)
	if !ok {
		return pkgauth.SupabaseUser{}, "", false
	}
	token, ok := GetAccessToken(c)
	if !ok {
		return pkgauth.SupabaseUser{}, "", false
	}
	return user, token, true
}
