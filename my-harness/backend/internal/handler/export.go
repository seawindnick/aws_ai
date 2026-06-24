package handler

import (
	"net/http"
	"path/filepath"

	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
	"github.com/workshop/wrong-question/internal/service"
)

type ExportHandler struct {
	svc *service.ExportService
}

func NewExportHandler(svc *service.ExportService) *ExportHandler {
	return &ExportHandler{svc: svc}
}

func (h *ExportHandler) ExportPDF(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		WriteError(w, apperr.ErrUnauthorized)
		return
	}

	subject := r.URL.Query().Get("subject")

	outPath, err := h.svc.ExportPDF(r.Context(), userID, subject)
	if err != nil {
		WriteError(w, err)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(outPath))
	w.Header().Set("Content-Type", "application/pdf")
	http.ServeFile(w, r, outPath)
}
