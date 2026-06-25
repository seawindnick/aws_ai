package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/workshop/wrong-question/internal/apperr"
	"github.com/workshop/wrong-question/internal/middleware"
)

// ServeImage serves images under imageDir, enforcing that the {userID} path
// segment matches the authenticated user. Prevents cross-user image access.
func ServeImage(imageDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authedUserID, ok := middleware.UserIDFromCtx(r.Context())
		if !ok {
			WriteError(w, apperr.ErrUnauthorized)
			return
		}

		pathUserID := chi.URLParam(r, "userID")
		if pathUserID != authedUserID {
			WriteError(w, apperr.ErrForbidden)
			return
		}

		// Strip "/images/{userID}" prefix, serve the rest from imageDir/{userID}/
		rest := chi.URLParam(r, "*")
		// Prevent path traversal
		if strings.Contains(rest, "..") {
			WriteError(w, apperr.BadRequest("invalid path"))
			return
		}

		// Rewrite URL so http.FileServer sees only the filename portion.
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/" + rest
		root := http.Dir(filepath.Join(imageDir, pathUserID))
		http.FileServer(root).ServeHTTP(w, r2)
	}
}
