package handler

import (
	"net/http"
	"strconv"

	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/service"
)

type ErrorRecordHandler struct {
	svc *service.ErrorRecordService
}

func NewErrorRecordHandler(svc *service.ErrorRecordService) *ErrorRecordHandler {
	return &ErrorRecordHandler{svc: svc}
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
	records, err := h.svc.List(r.Context(), userID, page, pageSize)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteList(w, records, len(records))
}
