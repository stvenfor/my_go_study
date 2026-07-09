// supabase_session_auth.go 校验 Supabase token + 单设备 session。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
	pkgauth "github.com/stvenfor/my_go_study/pkg/auth"
	"github.com/stvenfor/my_go_study/pkg/config"
)

const (
	HeaderSessionID = "X-Session-ID"
	HeaderDeviceID  = "X-Device-ID"
)

// SupabaseSessionAuth 校验 Supabase JWT 与 Redis 中的唯一 mobile session。
func SupabaseSessionAuth(cfg config.SupabaseConfig, sessionUC *usecase.DeviceSessionUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			response.BackendError(c, 401, "未提供 Authorization 头")
			c.Abort()
			return
		}

		user, err := pkgauth.ValidateAccessToken(c.Request.Context(), cfg, token)
		if err != nil {
			response.BackendError(c, 401, err.Error())
			c.Abort()
			return
		}

		sessionID := strings.TrimSpace(c.GetHeader(HeaderSessionID))
		deviceID := strings.TrimSpace(c.GetHeader(HeaderDeviceID))
		if sessionUC != nil {
			if err := sessionUC.Validate(c.Request.Context(), user.ID, user.Email, sessionID, deviceID); err != nil {
				switch {
				case err == usecase.ErrSessionReplaced:
					response.BackendError(c, 401, usecase.MsgSessionReplaced)
				default:
					response.BackendError(c, 401, usecase.MsgSessionInvalid)
				}
				c.Abort()
				return
			}
		}

		c.Set(ContextSupabaseUserKey, user)
		c.Set(ContextAccessTokenKey, token)
		c.Next()
	}
}
