package handler

import (
	"fmt"
	"net/http"

	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/repository"
	"github.com/workshop/wrong-question/internal/service"
)

type RecommendHandler struct {
	bedrockSvc  *service.BedrockService
	errorRepo   *repository.ErrorRecordRepo
}

func NewRecommendHandler(bedrockSvc *service.BedrockService, errorRepo *repository.ErrorRecordRepo) *RecommendHandler {
	return &RecommendHandler{bedrockSvc: bedrockSvc, errorRepo: errorRepo}
}

func (h *RecommendHandler) Recommend(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	errors, err := h.errorRepo.ListByUser(r.Context(), userID)
	if err != nil {
		WriteError(w, err)
		return
	}

	summary := fmt.Sprintf("User has %d error records.", len(errors))
	for _, e := range errors {
		summary += fmt.Sprintf("\n- question_id: %s, wrong_count: %d, last_wrong: %s",
			e.QuestionID, e.WrongCount, e.LastWrongAt.Format("2006-01-02"))
	}

	items, err := h.bedrockSvc.Recommend(r.Context(), summary)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteList(w, items, len(items))
}
