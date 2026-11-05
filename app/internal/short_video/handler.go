// handler.go 小视频 HTTP。
//
// 由 app/internal/delivery/http/controller/short_video_controller.go 迁入。
// 取身份改用 authctx.SupabaseAuthContext（middleware 会成环，见 access 域批 1 记录）；
// 错误映射改用 shared.WriteResourceCRUDError。
package short_video

import (
	"net/http"
	"strings"

	"app/internal/delivery/http/authctx"
	"app/internal/delivery/http/dto/response"
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// handler 小视频 HTTP。
type handler struct {
	svc *Service
}

func newHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

func (h *handler) Create(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body struct {
		Title       string   `json:"title"`
		VideoURL    *string  `json:"video_url"`
		CoverURL    *string  `json:"cover_url"`
		Duration    *string  `json:"duration"`
		AspectRatio *float64 `json:"aspect_ratio"`
		TopicID     *string  `json:"topic_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	dto, err := h.svc.Create(c.Request.Context(), user.ID, CreateShortVideoInput{
		Title: body.Title, VideoURL: body.VideoURL, CoverURL: body.CoverURL,
		Duration: body.Duration, AspectRatio: body.AspectRatio, TopicID: body.TopicID,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessCreated(c, dto)
}

func (h *handler) List(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 20)
	list, total, err := h.svc.List(
		c.Request.Context(),
		user.ID,
		strings.TrimSpace(c.Query("scope")),
		strings.TrimSpace(c.Query("user_id")),
		pq.Page, pq.Size,
	)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func (h *handler) Profile(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	dto, err := h.svc.Profile(c.Request.Context(), user.ID, strings.TrimSpace(c.Query("user_id")))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, dto)
}

func (h *handler) Get(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	dto, err := h.svc.Get(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, dto)
}

func (h *handler) Delete(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	if err := h.svc.SoftDelete(c.Request.Context(), user.ID, c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, nil)
}

func (h *handler) Like(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	dto, err := h.svc.SetLike(c.Request.Context(), user.ID, c.Param("id"), true)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{
		"id": dto.ID, "like_count": dto.LikeCount, "is_liked": dto.IsLiked,
	})
}

func (h *handler) Unlike(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	dto, err := h.svc.SetLike(c.Request.Context(), user.ID, c.Param("id"), false)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{
		"id": dto.ID, "like_count": dto.LikeCount, "is_liked": dto.IsLiked,
	})
}

func (h *handler) View(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	n, err := h.svc.AddView(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{"view_count": n})
}

// Review 人工放行（parity-p0-4）：body 只接受 {"action":"approve"}，权限在 Service 内校验。
func (h *handler) Review(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	if strings.ToLower(strings.TrimSpace(body.Action)) != "approve" {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "action 仅支持 approve")
		return
	}
	dto, err := h.svc.Approve(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, dto)
}

func writeError(c *gin.Context, err error) {
	shared.WriteResourceCRUDError(c, err, ErrInvalid, ErrNotFound, ErrForbidden)
}
