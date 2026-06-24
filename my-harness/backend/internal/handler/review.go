package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/service"
)

type ReviewHandler struct {
	svc *service.ReviewService
}

func NewReviewHandler(svc *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

func (h *ReviewHandler) Today(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	schedules, err := h.svc.TodayList(r.Context(), userID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteList(w, schedules, len(schedules))
}

func (h *ReviewHandler) SubmitResult(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
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
		Result string `json:"result"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}

	result := model.ReviewResult(body.Result)
	if result != model.ReviewPass && result != model.ReviewFail {
		WriteError(w, apperr.BadRequest("result must be 'pass' or 'fail'"))
		return
	}

	if err := h.svc.SubmitResult(r.Context(), userID, questionID, result); err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
