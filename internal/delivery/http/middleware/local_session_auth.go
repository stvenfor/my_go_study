// local_session_auth.go 校验本地 UUID JWT + 单设备 session。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
	pkgauth "github.com/stvenfor/my_go_study/pkg/auth"
	jwtmanager "github.com/stvenfor/my_go_study/pkg/jwt"
)

// LocalSessionAuth 校验本地 access JWT 与 Redis device session；上下文键与 Supabase 路径一致。
func LocalSessionAuth(jwtMgr *jwtmanager.Manager, sessionUC *usecase.DeviceSessionUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			response.BackendError(c, 401, "未提供 Authorization 头")
			c.Abort()
			return
		}

		claims, err := jwtMgr.ParseUUID(token)
		if err != nil {
			response.BackendError(c, 401, "token 无效")
			c.Abort()
			return
		}

		user := pkgauth.SupabaseUser{
			ID:    claims.Subject,
			Email: claims.Email,
		}

		sessionID := strings.TrimSpace(c.GetHeader(HeaderSessionID))
		deviceID := strings.TrimSpace(c.GetHeader(HeaderDeviceID))
		if sessionUC != nil {
			if err := sessionUC.Validate(c.Request.Context(), user.ID, user.Email, sessionID, deviceID); err != nil {
				abortDeviceSessionError(c, err)
				return
			}
		}

		c.Set(ContextSupabaseUserKey, user)
		c.Set(ContextAccessTokenKey, token)
		c.Next()
	}
}
