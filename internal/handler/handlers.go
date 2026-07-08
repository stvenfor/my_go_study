package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/stvenfor/my_go_study/internal/domain"
	"github.com/stvenfor/my_go_study/internal/httpx"
	"github.com/stvenfor/my_go_study/internal/middleware"
	"github.com/stvenfor/my_go_study/internal/repository"
)

type HealthHandler struct{}

func (h HealthHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "my_go_study",
	})
}

type ProfileHandler struct {
	repo *repository.ProfileRepository
}

func NewProfileHandler(repo *repository.ProfileRepository) *ProfileHandler {
	return &ProfileHandler{repo: repo}
}

func (h *ProfileHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.User(r.Context())
	profile, err := h.repo.GetByUserID(r.Context(), middleware.AccessToken(r.Context()), user.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, profile)
}

func (h *ProfileHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var input domain.UpdateProfileInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求体格式错误")
		return
	}

	user := middleware.User(r.Context())
	profile, err := h.repo.UpdateByUserID(r.Context(), middleware.AccessToken(r.Context()), user.ID, input)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, profile)
}

type TransactionHandler struct {
	repo *repository.TransactionRepository
}

func NewTransactionHandler(repo *repository.TransactionRepository) *TransactionHandler {
	return &TransactionHandler{repo: repo}
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.TransactionFilter{
		Type: r.URL.Query().Get("type"),
	}
	if limit, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
		filter.Limit = limit
	}
	if offset, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil {
		filter.Offset = offset
	}

	items, err := h.repo.List(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *TransactionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "无效的 id")
		return
	}

	item, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求体格式错误")
		return
	}
	if input.Type == "" || input.Category == "" || input.Date == "" {
		httpx.Error(w, http.StatusBadRequest, "type、category、date 为必填")
		return
	}

	item, err := h.repo.Create(r.Context(), input)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, item)
}

func (h *TransactionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "无效的 id")
		return
	}

	var input domain.UpdateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求体格式错误")
		return
	}

	item, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (h *TransactionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "无效的 id")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
