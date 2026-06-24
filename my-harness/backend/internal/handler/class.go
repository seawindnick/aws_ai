package handler

import (
	"encoding/json"
	"net/http"

	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/service"
)

type ClassHandler struct {
	svc *service.ClassService
}

func NewClassHandler(svc *service.ClassService) *ClassHandler {
	return &ClassHandler{svc: svc}
}

func (h *ClassHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromCtx(r.Context())
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		WriteError(w, apperr.BadRequest("name required"))
		return
	}
	c, err := h.svc.Create(r.Context(), userID, body.Name)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, c)
}

func (h *ClassHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromCtx(r.Context())
	role, _ := middleware.RoleFromCtx(r.Context())
	classes, err := h.svc.ListMy(r.Context(), userID, role)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteList(w, classes, len(classes))
}

func (h *ClassHandler) Detail(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromCtx(r.Context())
	role, _ := middleware.RoleFromCtx(r.Context())
	classID := r.URL.Query().Get("class_id")
	if classID == "" {
		WriteError(w, apperr.BadRequest("class_id required"))
		return
	}
	c, err := h.svc.Get(r.Context(), classID, userID, role)
	if err != nil {
		WriteError(w, apperr.NotFound(err.Error()))
		return
	}
	WriteJSON(w, http.StatusOK, c)
}

func (h *ClassHandler) Join(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromCtx(r.Context())
	var body struct {
		InviteCode string `json:"invite_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.InviteCode == "" {
		WriteError(w, apperr.BadRequest("invite_code required"))
		return
	}
	c, err := h.svc.Join(r.Context(), userID, body.InviteCode)
	if err != nil {
		WriteError(w, apperr.NotFound(err.Error()))
		return
	}
	WriteJSON(w, http.StatusOK, c)
}

func (h *ClassHandler) Leave(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromCtx(r.Context())
	var body struct {
		ClassID string `json:"class_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ClassID == "" {
		WriteError(w, apperr.BadRequest("class_id required"))
		return
	}
	if err := h.svc.Leave(r.Context(), body.ClassID, userID); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ClassHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromCtx(r.Context())
	var body struct {
		ClassID string `json:"class_id"`
		UserID  string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ClassID == "" || body.UserID == "" {
		WriteError(w, apperr.BadRequest("class_id and user_id required"))
		return
	}
	if err := h.svc.RemoveMember(r.Context(), body.ClassID, userID, body.UserID); err != nil {
		WriteError(w, apperr.Forbidden(err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ClassHandler) ResetCode(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromCtx(r.Context())
	var body struct {
		ClassID string `json:"class_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ClassID == "" {
		WriteError(w, apperr.BadRequest("class_id required"))
		return
	}
	code, err := h.svc.ResetCode(r.Context(), body.ClassID, userID)
	if err != nil {
		WriteError(w, apperr.Forbidden(err.Error()))
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"invite_code": code})
}
