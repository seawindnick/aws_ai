package handler

import (
	"net/http"
	"strconv"

	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/repository"
)

type ErrorRecordHandler struct {
	repo *repository.ErrorRecordRepo
}

func NewErrorRecordHandler(repo *repository.ErrorRecordRepo) *ErrorRecordHandler {
	return &ErrorRecordHandler{repo: repo}
}

func (h *ErrorRecordHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	records, err := h.repo.ListByUser(r.Context(), userID, page, pageSize)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteList(w, records, len(records))
}
