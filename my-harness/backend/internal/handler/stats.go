package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/service"
)

type StatsHandler struct {
	svc *service.StatsService
}

func NewStatsHandler(svc *service.StatsService) *StatsHandler {
	return &StatsHandler{svc: svc}
}

func (h *StatsHandler) MyStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}
	from, to, err := parseDateRange(r)
	if err != nil {
		WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	stats, err := h.svc.GetStudentStats(r.Context(), userID, from, to)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, stats)
}

func (h *StatsHandler) ClassStats(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	stats, err := h.svc.GetClassStats(r.Context(), from, to)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteList(w, stats, len(stats))
}

func (h *StatsHandler) StudentStats(w http.ResponseWriter, r *http.Request) {
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		WriteError(w, apperr.BadRequest("student_id required"))
		return
	}
	from, to, err := parseDateRange(r)
	if err != nil {
		WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	stats, err := h.svc.GetStudentStatsByID(r.Context(), studentID, from, to)
	if err != nil {
		msg := err.Error()
		if msg == "student not found" || msg == "user is not a student" {
			WriteError(w, apperr.NotFound(msg))
		} else {
			WriteError(w, err)
		}
		return
	}
	WriteJSON(w, http.StatusOK, stats)
}

func parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	q := r.URL.Query()
	fromStr := q.Get("date_from")
	toStr := q.Get("date_to")

	var from, to time.Time
	var err error

	if fromStr != "" {
		from, err = time.Parse(time.DateOnly, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("date_from must be YYYY-MM-DD")
		}
	} else {
		from = time.Now().UTC().AddDate(0, -1, 0)
	}
	if toStr != "" {
		to, err = time.Parse(time.DateOnly, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("date_to must be YYYY-MM-DD")
		}
	} else {
		to = time.Now().UTC()
	}
	return from, to, nil
}
