package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/service"
)

// ReviewQueueHandler 处理人工审核队列，挂载在需要 teacher/admin 角色的路由组下。
type ReviewQueueHandler struct {
	svc *service.ReviewQueueService
}

func NewReviewQueueHandler(svc *service.ReviewQueueService) *ReviewQueueHandler {
	return &ReviewQueueHandler{svc: svc}
}

func (h *ReviewQueueHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	questions, err := h.svc.ListPending(r.Context(), page, pageSize)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteList(w, questions, len(questions))
}

func (h *ReviewQueueHandler) Review(w http.ResponseWriter, r *http.Request) {
	reviewerID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	questionID := chi.URLParam(r, "id")
	if questionID == "" {
		WriteError(w, apperr.BadRequest("question id required"))
		return
	}

	var body struct {
		Action   string `json:"action"`   // "approve" | "reject"
		Category string `json:"category"` // 可选，修正分类
		Note     string `json:"note"`     // 可选，审核备注
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if body.Action == "" {
		WriteError(w, apperr.BadRequest("action required: 'approve' or 'reject'"))
		return
	}

	action := model.ReviewAction(body.Action)
	category := model.QuestionCategory(body.Category)

	if err := h.svc.Review(r.Context(), questionID, reviewerID, action, category, body.Note); err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
