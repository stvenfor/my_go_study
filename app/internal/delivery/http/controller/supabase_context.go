// supabase_context.go 兼容转发层。
//
// 真实实现已上提到 middleware.SupabaseAuthContext（域切片后各域直接调用）。
// 本文件只为尚未迁移的 controller 保留同名函数，避免一次性改动 20 个文件。
// 待 批 2-4 把剩余 controller 迁入各域后，直接删除本文件。
package controller

import (
	"app/internal/delivery/http/middleware"
	pkgauth "common/auth"
	"github.com/gin-gonic/gin"
)

func supabaseAuthContext(c *gin.Context) (pkgauth.SupabaseUser, string, bool) {
	return middleware.SupabaseAuthContext(c)
}
