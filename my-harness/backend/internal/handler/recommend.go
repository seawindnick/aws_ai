package handler

import (
	"net/http"

	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/service"
)

type RecommendHandler struct {
	svc *service.RecommendService
}

func NewRecommendHandler(svc *service.RecommendService) *RecommendHandler {
	return &RecommendHandler{svc: svc}
}

func (h *RecommendHandler) Recommend(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	items, err := h.svc.Recommend(r.Context(), userID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteList(w, items, len(items))
}
