package handler

import (
	"net/http"

	"github.com/workshop/wrong-question/internal/httputil"
)

// WriteJSON, WriteError, WriteList delegate to httputil so handler code stays
// unchanged while middleware can import httputil directly (no import cycle).

func WriteJSON(w http.ResponseWriter, code int, body any) {
	httputil.WriteJSON(w, code, body)
}

func WriteError(w http.ResponseWriter, err error) {
	httputil.WriteError(w, err)
}

func WriteList(w http.ResponseWriter, items any, count int) {
	httputil.WriteList(w, items, count)
}
