package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// ShortVideoController 小视频 HTTP。
type ShortVideoController struct {
	uc *usecase.ShortVideoUsecase
}

func NewShortVideoController(uc *usecase.ShortVideoUsecase) *ShortVideoController {
	return &ShortVideoController{uc: uc}
}

func (ctrl *ShortVideoController) Create(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
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
	dto, err := ctrl.uc.Create(c.Request.Context(), user.ID, usecase.CreateShortVideoInput{
		Title: body.Title, VideoURL: body.VideoURL, CoverURL: body.CoverURL,
		Duration: body.Duration, AspectRatio: body.AspectRatio, TopicID: body.TopicID,
	})
	if err != nil {
		writeShortVideoError(c, err)
		return
	}
	response.SuccessCreated(c, dto)
}

func (ctrl *ShortVideoController) List(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 20)
	list, total, err := ctrl.uc.List(
		c.Request.Context(),
		user.ID,
		strings.TrimSpace(c.Query("scope")),
		strings.TrimSpace(c.Query("user_id")),
		pq.Page, pq.Size,
	)
	if err != nil {
		writeShortVideoError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func (ctrl *ShortVideoController) Profile(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	dto, err := ctrl.uc.Profile(c.Request.Context(), user.ID, strings.TrimSpace(c.Query("user_id")))
	if err != nil {
		writeShortVideoError(c, err)
		return
	}
	response.Success(c, dto)
}

func (ctrl *ShortVideoController) Get(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	dto, err := ctrl.uc.Get(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		writeShortVideoError(c, err)
		return
	}
	response.Success(c, dto)
}

func (ctrl *ShortVideoController) Delete(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	if err := ctrl.uc.SoftDelete(c.Request.Context(), user.ID, c.Param("id")); err != nil {
		writeShortVideoError(c, err)
		return
	}
	response.Success(c, nil)
}

func (ctrl *ShortVideoController) Like(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	dto, err := ctrl.uc.SetLike(c.Request.Context(), user.ID, c.Param("id"), true)
	if err != nil {
		writeShortVideoError(c, err)
		return
	}
	response.Success(c, gin.H{
		"id": dto.ID, "like_count": dto.LikeCount, "is_liked": dto.IsLiked,
	})
}

func (ctrl *ShortVideoController) Unlike(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	dto, err := ctrl.uc.SetLike(c.Request.Context(), user.ID, c.Param("id"), false)
	if err != nil {
		writeShortVideoError(c, err)
		return
	}
	response.Success(c, gin.H{
		"id": dto.ID, "like_count": dto.LikeCount, "is_liked": dto.IsLiked,
	})
}

func (ctrl *ShortVideoController) View(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	n, err := ctrl.uc.AddView(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		writeShortVideoError(c, err)
		return
	}
	response.Success(c, gin.H{"view_count": n})
}

func writeShortVideoError(c *gin.Context, err error) {
	writeResourceCRUDError(c, err, usecase.ErrShortVideoInvalid, usecase.ErrShortVideoNotFound, usecase.ErrShortVideoForbidden)
}
