package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/service"
)

type PaperHandler struct {
	svc    *service.PaperService
	export *service.ExportService
}

func NewPaperHandler(svc *service.PaperService, export *service.ExportService) *PaperHandler {
	return &PaperHandler{svc: svc, export: export}
}

// Create POST /api/papers
func (h *PaperHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if body.Title == "" {
		WriteError(w, apperr.BadRequest("title required"))
		return
	}

	paper, err := h.svc.Create(r.Context(), userID, body.Title)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, paper)
}

// List GET /api/papers
func (h *PaperHandler) List(w http.ResponseWriter, r *http.Request) {
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

	papers, err := h.svc.List(r.Context(), userID, page, pageSize)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteList(w, papers, len(papers))
}

// Get GET /api/papers/{id}
func (h *PaperHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		WriteError(w, apperr.BadRequest("id required"))
		return
	}

	paper, err := h.svc.Get(r.Context(), id, userID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, paper)
}

// Rename PATCH /api/papers/{id}
func (h *PaperHandler) Rename(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		WriteError(w, apperr.BadRequest("id required"))
		return
	}

	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if body.Title == "" {
		WriteError(w, apperr.BadRequest("title required"))
		return
	}

	if err := h.svc.Rename(r.Context(), id, userID, body.Title); err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete DELETE /api/papers/{id}
func (h *PaperHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		WriteError(w, apperr.BadRequest("id required"))
		return
	}

	if err := h.svc.Delete(r.Context(), id, userID); err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddQuestion POST /api/papers/{id}/questions
func (h *PaperHandler) AddQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	paperID := chi.URLParam(r, "id")
	if paperID == "" {
		WriteError(w, apperr.BadRequest("paper id required"))
		return
	}

	var body struct {
		QuestionID string `json:"question_id"`
		Position   int    `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if body.QuestionID == "" {
		WriteError(w, apperr.BadRequest("question_id required"))
		return
	}

	if err := h.svc.AddQuestion(r.Context(), paperID, body.QuestionID, userID, body.Position); err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveQuestion DELETE /api/papers/{id}/questions/{qid}
func (h *PaperHandler) RemoveQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	paperID := chi.URLParam(r, "id")
	questionID := chi.URLParam(r, "qid")
	if paperID == "" || questionID == "" {
		WriteError(w, apperr.BadRequest("paper id and question id required"))
		return
	}

	if err := h.svc.RemoveQuestion(r.Context(), paperID, questionID, userID); err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Reorder PUT /api/papers/{id}/reorder
func (h *PaperHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	paperID := chi.URLParam(r, "id")
	if paperID == "" {
		WriteError(w, apperr.BadRequest("paper id required"))
		return
	}

	var body struct {
		Positions []struct {
			QuestionID string `json:"question_id"`
			Position   int    `json:"position"`
		} `json:"positions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if len(body.Positions) == 0 {
		WriteError(w, apperr.BadRequest("positions required"))
		return
	}

	positions := make([]struct {
		QuestionID string
		Position   int
	}, len(body.Positions))
	for i, p := range body.Positions {
		if p.QuestionID == "" {
			WriteError(w, apperr.BadRequest("each position entry must have question_id"))
			return
		}
		positions[i] = struct {
			QuestionID string
			Position   int
		}{QuestionID: p.QuestionID, Position: p.Position}
	}

	if err := h.svc.Reorder(r.Context(), paperID, userID, positions); err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListQuestions GET /api/papers/{id}/questions
func (h *PaperHandler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	paperID := chi.URLParam(r, "id")
	if paperID == "" {
		WriteError(w, apperr.BadRequest("paper id required"))
		return
	}

	items, err := h.svc.ListQuestions(r.Context(), paperID, userID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteList(w, items, len(items))
}

// Duplicate POST /api/papers/{id}/duplicate
func (h *PaperHandler) Duplicate(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	paperID := chi.URLParam(r, "id")
	if paperID == "" {
		WriteError(w, apperr.BadRequest("paper id required"))
		return
	}

	paper, err := h.svc.Duplicate(r.Context(), paperID, userID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, paper)
}

// ExportPDF POST /api/papers/{id}/export
func (h *PaperHandler) ExportPDF(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	paperID := chi.URLParam(r, "id")
	if paperID == "" {
		WriteError(w, apperr.BadRequest("paper id required"))
		return
	}

	path, skipped, err := h.export.ExportPaper(r.Context(), paperID, userID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"path":    path,
		"skipped": skipped,
	})
}

// Download GET /api/papers/download?id=
func (h *PaperHandler) Download(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	paperID := r.URL.Query().Get("id")
	if paperID == "" {
		WriteError(w, apperr.BadRequest("id required"))
		return
	}

	filePath, err := h.export.GetPaperDownloadPath(r.Context(), paperID, userID)
	if err != nil {
		WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+paperID+".pdf\"")
	http.ServeFile(w, r, filePath)
}
