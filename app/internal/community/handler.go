package community

import (
	"net/http"
	"strings"

	"app/internal/delivery/http/authctx"
	"app/internal/delivery/http/dto/response"
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// handler 社区动态 HTTP。
type handler struct {
	svc *Service
}

func newHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

func (h *handler) ListTopics(c *gin.Context) {
	if _, _, ok := authctx.SupabaseAuthContext(c); !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 20)
	list, total, err := h.svc.ListTopics(c.Request.Context(), c.Query("q"), pq.Page, pq.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "获取话题失败")
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func (h *handler) SearchTopics(c *gin.Context) {
	h.ListTopics(c)
}

func (h *handler) Search(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 10)
	typ := c.Query("type")
	data, total, err := h.svc.Search(c.Request.Context(), user.ID, c.Query("q"), typ, pq.Page, pq.Size)
	if err != nil {
		writeError(c, err)
		return
	}
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "post", "posts", "动态":
		list, _ := data.([]PostDTO)
		response.SuccessList(c, list, pq.Page, pq.Size, total)
	case "topic", "topics", "话题":
		list, _ := data.([]TopicDTO)
		response.SuccessList(c, list, pq.Page, pq.Size, total)
	case "user", "users", "用户":
		list, _ := data.([]CommunityUserDTO)
		response.SuccessList(c, list, pq.Page, pq.Size, total)
	default:
		response.Success(c, data)
	}
}

func (h *handler) CreatePost(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
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
	post, err := h.svc.CreatePost(c.Request.Context(), user.ID, CreatePostInput{
		Content: body.Content, MediaType: body.MediaType, ImageURLs: body.ImageURLs,
		VideoURL: body.VideoURL, VideoCoverURL: body.VideoCoverURL,
		TopicID: body.TopicID, IsAskEveryone: body.IsAskEveryone, Source: body.Source,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessCreated(c, post)
}

func (h *handler) ListPosts(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 10)
	list, total, err := h.svc.ListPosts(c.Request.Context(), user.ID, c.Query("tab"), pq.Page, pq.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "获取动态失败")
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func (h *handler) FollowUser(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	if err := h.svc.FollowUser(c.Request.Context(), user.ID, c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{"followee_id": c.Param("id"), "is_followed": true})
}

func (h *handler) UnfollowUser(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	if err := h.svc.UnfollowUser(c.Request.Context(), user.ID, c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{"followee_id": c.Param("id"), "is_followed": false})
}

func (h *handler) DeletePost(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	if err := h.svc.SoftDelete(c.Request.Context(), user.ID, c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{})
}

func (h *handler) LikePost(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	post, err := h.svc.SetLike(c.Request.Context(), user.ID, c.Param("id"), true)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{
		"id": post.ID, "like_count": post.LikeCount, "is_liked": post.IsLiked,
	})
}

func (h *handler) UnlikePost(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	post, err := h.svc.SetLike(c.Request.Context(), user.ID, c.Param("id"), false)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{
		"id": post.ID, "like_count": post.LikeCount, "is_liked": post.IsLiked,
	})
}

func (h *handler) ListComments(c *gin.Context) {
	if _, _, ok := authctx.SupabaseAuthContext(c); !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 20)
	list, total, err := h.svc.ListComments(c.Request.Context(), c.Param("id"), pq.Page, pq.Size)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func (h *handler) AddComment(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
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
	comment, err := h.svc.AddComment(c.Request.Context(), user.ID, c.Param("id"), body.Content, body.ReplyToNickname)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessCreated(c, comment)
}

func writeError(c *gin.Context, err error) {
	shared.WriteResourceCRUDError(c, err, ErrInvalid, ErrNotFound, ErrForbidden)
}
