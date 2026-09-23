package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// CommunityController 社区动态 HTTP。
type CommunityController struct {
	uc *usecase.CommunityUsecase
}

func NewCommunityController(uc *usecase.CommunityUsecase) *CommunityController {
	return &CommunityController{uc: uc}
}

func (ctrl *CommunityController) ListTopics(c *gin.Context) {
	if _, _, ok := supabaseAuthContext(c); !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 20)
	list, total, err := ctrl.uc.ListTopics(c.Request.Context(), c.Query("q"), pq.Page, pq.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "获取话题失败")
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func (ctrl *CommunityController) SearchTopics(c *gin.Context) {
	ctrl.ListTopics(c)
}

func (ctrl *CommunityController) Search(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 10)
	typ := c.Query("type")
	data, total, err := ctrl.uc.Search(c.Request.Context(), user.ID, c.Query("q"), typ, pq.Page, pq.Size)
	if err != nil {
		writeCommunityError(c, err)
		return
	}
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "post", "posts", "动态":
		list, _ := data.([]usecase.PostDTO)
		response.SuccessList(c, list, pq.Page, pq.Size, total)
	case "topic", "topics", "话题":
		list, _ := data.([]usecase.TopicDTO)
		response.SuccessList(c, list, pq.Page, pq.Size, total)
	case "user", "users", "用户":
		list, _ := data.([]usecase.CommunityUserDTO)
		response.SuccessList(c, list, pq.Page, pq.Size, total)
	default:
		response.Success(c, data)
	}
}

func (ctrl *CommunityController) CreatePost(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body struct {
		Content       string   `json:"content"`
		MediaType     string   `json:"media_type"`
		ImageURLs     []string `json:"image_urls"`
		VideoURL      *string  `json:"video_url"`
		VideoCoverURL *string  `json:"video_cover_url"`
		TopicID       *string  `json:"topic_id"`
		IsAskEveryone bool     `json:"is_ask_everyone"`
		Source        string   `json:"source"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	post, err := ctrl.uc.CreatePost(c.Request.Context(), user.ID, usecase.CreatePostInput{
		Content: body.Content, MediaType: body.MediaType, ImageURLs: body.ImageURLs,
		VideoURL: body.VideoURL, VideoCoverURL: body.VideoCoverURL,
		TopicID: body.TopicID, IsAskEveryone: body.IsAskEveryone, Source: body.Source,
	})
	if err != nil {
		writeCommunityError(c, err)
		return
	}
	response.SuccessCreated(c, post)
}

func (ctrl *CommunityController) ListPosts(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 10)
	list, total, err := ctrl.uc.ListPosts(c.Request.Context(), user.ID, c.Query("tab"), pq.Page, pq.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "获取动态失败")
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func (ctrl *CommunityController) FollowUser(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	if err := ctrl.uc.FollowUser(c.Request.Context(), user.ID, c.Param("id")); err != nil {
		writeCommunityError(c, err)
		return
	}
	response.Success(c, gin.H{"followee_id": c.Param("id"), "is_followed": true})
}

func (ctrl *CommunityController) UnfollowUser(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	if err := ctrl.uc.UnfollowUser(c.Request.Context(), user.ID, c.Param("id")); err != nil {
		writeCommunityError(c, err)
		return
	}
	response.Success(c, gin.H{"followee_id": c.Param("id"), "is_followed": false})
}

func (ctrl *CommunityController) DeletePost(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	if err := ctrl.uc.SoftDelete(c.Request.Context(), user.ID, c.Param("id")); err != nil {
		writeCommunityError(c, err)
		return
	}
	response.Success(c, gin.H{})
}

func (ctrl *CommunityController) LikePost(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	post, err := ctrl.uc.SetLike(c.Request.Context(), user.ID, c.Param("id"), true)
	if err != nil {
		writeCommunityError(c, err)
		return
	}
	response.Success(c, gin.H{
		"id": post.ID, "like_count": post.LikeCount, "is_liked": post.IsLiked,
	})
}

func (ctrl *CommunityController) UnlikePost(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	post, err := ctrl.uc.SetLike(c.Request.Context(), user.ID, c.Param("id"), false)
	if err != nil {
		writeCommunityError(c, err)
		return
	}
	response.Success(c, gin.H{
		"id": post.ID, "like_count": post.LikeCount, "is_liked": post.IsLiked,
	})
}

func (ctrl *CommunityController) ListComments(c *gin.Context) {
	if _, _, ok := supabaseAuthContext(c); !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 20)
	list, total, err := ctrl.uc.ListComments(c.Request.Context(), c.Param("id"), pq.Page, pq.Size)
	if err != nil {
		writeCommunityError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func (ctrl *CommunityController) AddComment(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body struct {
		Content         string  `json:"content"`
		ReplyToNickname *string `json:"reply_to_nickname"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	comment, err := ctrl.uc.AddComment(c.Request.Context(), user.ID, c.Param("id"), body.Content, body.ReplyToNickname)
	if err != nil {
		writeCommunityError(c, err)
		return
	}
	response.SuccessCreated(c, comment)
}

func writeCommunityError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrCommunityInvalid):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	case errors.Is(err, usecase.ErrCommunityNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "资源不存在")
	case errors.Is(err, usecase.ErrCommunityForbidden):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "无权限")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "操作失败")
	}
}
