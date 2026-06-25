package httputil

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/workshop/wrong-question/internal/apperr"
)

func WriteJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("write json response", "error", err)
	}
}

func WriteError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	msg := "internal server error"

	if e, ok := err.(*apperr.AppError); ok {
		code = e.Code
		msg = e.Message
	} else {
		slog.Error("unhandled error", "error", err)
	}

	WriteJSON(w, code, map[string]string{"error": msg})
}

func WriteList(w http.ResponseWriter, items any, count int) {
	WriteJSON(w, http.StatusOK, map[string]any{"items": items, "count": count})
}
