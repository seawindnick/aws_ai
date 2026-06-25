package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/service"
)

type AdminHandler struct {
	svc         *service.AdminService
	questionSvc *service.QuestionService
}

func NewAdminHandler(svc *service.AdminService, questionSvc *service.QuestionService) *AdminHandler {
	return &AdminHandler{svc: svc, questionSvc: questionSvc}
}

func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if body.Email == "" || body.Role == "" {
		WriteError(w, apperr.BadRequest("email and role required"))
		return
	}
	role := model.Role(body.Role)
	if role != model.RoleStudent && role != model.RoleTeacher && role != model.RoleAdmin {
		WriteError(w, apperr.BadRequest("role must be student, teacher, or admin"))
		return
	}
	u, tempPwd, err := h.svc.CreateUser(r.Context(), body.Email, role)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{
		"user":             u,
		"initial_password": tempPwd,
	})
}

func (h *AdminHandler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		WriteError(w, apperr.BadRequest("invalid multipart form"))
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		WriteError(w, apperr.BadRequest("file field required"))
		return
	}
	defer file.Close()

	result, err := h.svc.ImportCSV(r.Context(), file)
	if err != nil {
		WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	WriteJSON(w, http.StatusOK, result)
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	users, err := h.svc.ListUsers(r.Context(), q.Get("role"), q.Get("status"), page, pageSize)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteList(w, users, len(users))
}

func (h *AdminHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"user_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	status := model.UserStatus(body.Status)
	if status != model.UserStatusActive && status != model.UserStatusInactive {
		WriteError(w, apperr.BadRequest("status must be active or inactive"))
		return
	}
	if err := h.svc.SetStatus(r.Context(), body.UserID, status); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	role := model.Role(body.Role)
	if role != model.RoleStudent && role != model.RoleTeacher && role != model.RoleAdmin {
		WriteError(w, apperr.BadRequest("role must be student, teacher, or admin"))
		return
	}
	if err := h.svc.SetRole(r.Context(), body.UserID, role); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) ListUserQuestions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID := q.Get("user_id")
	if userID == "" {
		WriteError(w, apperr.BadRequest("user_id required"))
		return
	}
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	// admin context: pass "admin" role so status filter is not restricted to approved-only
	questions, err := h.questionSvc.List(r.Context(), userID, "", "", page, pageSize, string(model.RoleAdmin))
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteList(w, questions, len(questions))
}
