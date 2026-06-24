package handler

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/service"
)

type QuestionHandler struct {
	svc *service.QuestionService
}

func NewQuestionHandler(svc *service.QuestionService) *QuestionHandler {
	return &QuestionHandler{svc: svc}
}

func (h *QuestionHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		WriteError(w, apperr.BadRequest("invalid multipart form"))
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		WriteError(w, apperr.BadRequest("image field required"))
		return
	}
	defer file.Close()

	result, err := h.svc.Upload(r.Context(), userID, file)
	if err != nil {
		WriteError(w, err)
		return
	}

	code := http.StatusCreated
	if result.NeedsReview {
		code = http.StatusAccepted // 202: 已接收，待人工审核
	}
	WriteJSON(w, code, map[string]any{
		"question":       result.Question,
		"status_message": result.StatusMessage,
		"needs_review":   result.NeedsReview,
	})
}

func (h *QuestionHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	subject := r.URL.Query().Get("subject")
	status := model.QuestionStatus(r.URL.Query().Get("status"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	questions, err := h.svc.List(r.Context(), userID, subject, status, page, pageSize)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteList(w, questions, len(questions))
}

func (h *QuestionHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	q, err := h.svc.Get(r.Context(), id, userID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, q)
}

func (h *QuestionHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

// BatchUpload POST /api/questions/batch  (REQ-UPLOAD-10)
func (h *QuestionHandler) BatchUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		WriteError(w, apperr.BadRequest("invalid multipart form"))
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		WriteError(w, apperr.BadRequest("images field required"))
		return
	}

	multipartFiles := make([]multipart.File, 0, len(files))
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			WriteError(w, apperr.BadRequest("cannot open uploaded file"))
			return
		}
		defer f.Close()
		multipartFiles = append(multipartFiles, f)
	}

	result := h.svc.BatchUpload(r.Context(), userID, multipartFiles)
	WriteJSON(w, http.StatusMultiStatus, result)
}

// Search GET /api/questions/search  (REQ-SEARCH-01~06)
func (h *QuestionHandler) Search(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))

	var dateFrom, dateTo *time.Time
	if s := q.Get("date_from"); s != "" {
		t, err := time.Parse(time.DateOnly, s)
		if err != nil {
			WriteError(w, apperr.BadRequest("date_from must be YYYY-MM-DD"))
			return
		}
		dateFrom = &t
	}
	if s := q.Get("date_to"); s != "" {
		t, err := time.Parse(time.DateOnly, s)
		if err != nil {
			WriteError(w, apperr.BadRequest("date_to must be YYYY-MM-DD"))
			return
		}
		dateTo = &t
	}

	params := model.SearchParams{
		UserID:   userID,
		Subject:  q.Get("subject"),
		Tag:      q.Get("tag"),
		Keyword:  q.Get("keyword"),
		Category: model.QuestionCategory(q.Get("category")),
		Status:   model.QuestionStatus(q.Get("status")),
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Page:     page,
		PageSize: pageSize,
	}

	questions, err := h.svc.Search(r.Context(), params)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteList(w, questions, len(questions))
}

func (h *QuestionHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
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
		Category string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if body.Category == "" {
		WriteError(w, apperr.BadRequest("category required"))
		return
	}

	if err := h.svc.UpdateCategory(r.Context(), id, userID, model.QuestionCategory(body.Category)); err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
