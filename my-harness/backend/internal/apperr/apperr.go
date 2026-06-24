package apperr

import "net/http"

type AppError struct {
	Code    int
	Message string
}

func (e *AppError) Error() string { return e.Message }

func New(code int, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

var (
	ErrUnauthorized = New(http.StatusUnauthorized, "unauthorized")
	ErrForbidden    = New(http.StatusForbidden, "forbidden")
	ErrNotFound     = New(http.StatusNotFound, "not found")
	ErrBadRequest   = New(http.StatusBadRequest, "bad request")
	ErrInternal     = New(http.StatusInternalServerError, "internal server error")
)

func BadRequest(msg string) *AppError  { return New(http.StatusBadRequest, msg) }
func NotFound(msg string) *AppError    { return New(http.StatusNotFound, msg) }
func Forbidden(msg string) *AppError   { return New(http.StatusForbidden, msg) }
func Internal(msg string) *AppError    { return New(http.StatusInternalServerError, msg) }
