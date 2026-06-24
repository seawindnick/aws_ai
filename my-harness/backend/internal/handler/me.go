package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/service"
)

type MeHandler struct {
	svc *service.MeService
}

func NewMeHandler(svc *service.MeService) *MeHandler {
	return &MeHandler{svc: svc}
}

func (h *MeHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}
	u, err := h.svc.GetProfile(r.Context(), userID)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, u)
}

func (h *MeHandler) UpdateNickname(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}
	var body struct {
		Nickname string `json:"nickname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if err := h.svc.UpdateNickname(r.Context(), userID, body.Nickname); err != nil {
		WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *MeHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	var body struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if body.OldPassword == "" || body.NewPassword == "" {
		WriteError(w, apperr.BadRequest("old_password and new_password required"))
		return
	}
	if err := h.svc.ChangePassword(r.Context(), accessToken, body.OldPassword, body.NewPassword); err != nil {
		WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *MeHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}
	authHeader := r.Header.Get("Authorization")
	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if body.Password == "" {
		WriteError(w, apperr.BadRequest("password required"))
		return
	}
	if err := h.svc.Deactivate(r.Context(), userID, accessToken, body.Password); err != nil {
		WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
