package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/service"
)

type TagHandler struct {
	svc *service.TagService
}

func NewTagHandler(svc *service.TagService) *TagHandler {
	return &TagHandler{svc: svc}
}

// List GET /api/questions/{id}/tags
func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
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

	tags, err := h.svc.List(r.Context(), questionID, userID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteList(w, tags, len(tags))
}

// AddManual POST /api/questions/{id}/tags
func (h *TagHandler) AddManual(w http.ResponseWriter, r *http.Request) {
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
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if body.Name == "" {
		WriteError(w, apperr.BadRequest("name required"))
		return
	}

	tag, err := h.svc.AddManual(r.Context(), questionID, userID, body.Name)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, tag)
}

// Confirm POST /api/questions/{id}/tags/{tid}/confirm
func (h *TagHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	tagID := chi.URLParam(r, "tid")
	if tagID == "" {
		WriteError(w, apperr.BadRequest("tag id required"))
		return
	}

	if err := h.svc.Confirm(r.Context(), tagID, userID); err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete DELETE /api/questions/{id}/tags/{tid}
func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	tagID := chi.URLParam(r, "tid")
	if tagID == "" {
		WriteError(w, apperr.BadRequest("tag id required"))
		return
	}

	if err := h.svc.Delete(r.Context(), tagID, userID); err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
