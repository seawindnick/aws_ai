package handler

import (
	"encoding/json"
	"net/http"

	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/service"
)

type TaskHandler struct {
	svc *service.TaskService
}

func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromCtx(r.Context())
	var body struct {
		ClassID string  `json:"class_id"`
		PaperID string  `json:"paper_id"`
		Title   string  `json:"title"`
		DueAt   *string `json:"due_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}
	if body.ClassID == "" || body.PaperID == "" || body.Title == "" {
		WriteError(w, apperr.BadRequest("class_id, paper_id, and title required"))
		return
	}
	var dueAt interface{} = nil
	_ = dueAt
	t, err := h.svc.Create(r.Context(), userID, body.ClassID, body.PaperID, body.Title, nil)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, t)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	classID := r.URL.Query().Get("class_id")
	if classID == "" {
		WriteError(w, apperr.BadRequest("class_id required"))
		return
	}
	tasks, err := h.svc.List(r.Context(), classID)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteList(w, tasks, len(tasks))
}

func (h *TaskHandler) Detail(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		WriteError(w, apperr.BadRequest("task_id required"))
		return
	}
	t, err := h.svc.Get(r.Context(), taskID)
	if err != nil {
		WriteError(w, apperr.NotFound("task not found"))
		return
	}
	WriteJSON(w, http.StatusOK, t)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromCtx(r.Context())
	var body struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TaskID == "" {
		WriteError(w, apperr.BadRequest("task_id required"))
		return
	}
	status := model.ClassTaskStatus(body.Status)
	if status == "" {
		status = model.ClassTaskStatusActive
	}
	if err := h.svc.Update(r.Context(), body.TaskID, userID, nil, status); err != nil {
		msg := err.Error()
		if msg == "closed task cannot be updated" {
			WriteError(w, apperr.New(409, msg))
		} else if msg == "forbidden" {
			WriteError(w, apperr.ErrForbidden)
		} else {
			WriteError(w, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) Submit(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromCtx(r.Context())
	var body struct {
		TaskID  string `json:"task_id"`
		Results []struct {
			QuestionID string `json:"question_id"`
			Result     string `json:"result"`
		} `json:"results"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TaskID == "" {
		WriteError(w, apperr.BadRequest("invalid request body"))
		return
	}

	items := make([]struct {
		QuestionID string
		Result     model.ReviewResult
	}, 0, len(body.Results))
	for _, item := range body.Results {
		r := model.ReviewResult(item.Result)
		if r != model.ReviewPass && r != model.ReviewFail {
			WriteError(w, apperr.BadRequest("result must be pass or fail"))
			return
		}
		items = append(items, struct {
			QuestionID string
			Result     model.ReviewResult
		}{QuestionID: item.QuestionID, Result: r})
	}

	result, err := h.svc.Submit(r.Context(), userID, body.TaskID, items)
	if err != nil {
		msg := err.Error()
		if msg == "task is closed" || msg == "task due date has passed" {
			WriteError(w, apperr.Forbidden(msg))
		} else {
			WriteError(w, err)
		}
		return
	}
	WriteJSON(w, http.StatusMultiStatus, result)
}

func (h *TaskHandler) Progress(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromCtx(r.Context())
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		WriteError(w, apperr.BadRequest("task_id required"))
		return
	}
	submissions, err := h.svc.Progress(r.Context(), taskID, userID)
	if err != nil {
		WriteError(w, apperr.Forbidden(err.Error()))
		return
	}
	WriteList(w, submissions, len(submissions))
}
