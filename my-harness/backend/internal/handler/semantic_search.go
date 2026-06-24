package handler

import (
	"net/http"
	"strings"

	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/service"
)

type SemanticSearchHandler struct {
	svc *service.SemanticSearchService
}

func NewSemanticSearchHandler(svc *service.SemanticSearchService) *SemanticSearchHandler {
	return &SemanticSearchHandler{svc: svc}
}

func (h *SemanticSearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		WriteError(w, apperr.BadRequest("q required"))
		return
	}
	if len(q) > 500 {
		WriteError(w, apperr.BadRequest("q must be at most 500 characters"))
		return
	}

	subject := r.URL.Query().Get("subject")

	results, err := h.svc.Search(r.Context(), userID, q, subject)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "embedding failed") {
			WriteError(w, apperr.New(502, "embedding service unavailable"))
		} else if strings.Contains(msg, "query must be") {
			WriteError(w, apperr.BadRequest(msg))
		} else {
			WriteError(w, apperr.New(502, "semantic search unavailable"))
		}
		return
	}

	WriteList(w, results, len(results))
}
