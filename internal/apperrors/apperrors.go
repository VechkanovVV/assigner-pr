// Package apperrors содержит определения кодов ошибок.
package apperrors

import (
	"fmt"
	"net/http"
)

// Code - машинный код ошибки.
type Code string

// AppError представляет ошибку.
type AppError struct {
	Code    Code
	Message string
}

// Error реализует error.
func (e *AppError) Error() string {
	if e == nil {
		return "<nil>"
	}

	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// HTTPStatus возвращает подходящий HTTP статус для кода ошибки.
func (e *AppError) HTTPStatus() int {
	if s, ok := statusByCode[e.Code]; ok {
		return s
	}
	return http.StatusInternalServerError
}

// Коды ошибок
const (
	ErrTeamExists    Code = "TEAM_EXISTS"
	ErrPRExists      Code = "PR_EXISTS"
	ErrPRMerged      Code = "PR_MERGED"
	ErrNotAssigned   Code = "NOT_ASSIGNED"
	ErrNoCandidate   Code = "NO_CANDIDATE"
	ErrNotFound      Code = "NOT_FOUND"
	ErrInternalIssue Code = "INTERNAL_ISSUE"
)

// messages - человекочитаемые строки по коду.
var messages = map[Code]string{
	ErrTeamExists:    "team_name already exists",
	ErrPRExists:      "PR id already exists",
	ErrPRMerged:      "cannot reassign on merged PR",
	ErrNotAssigned:   "reviewer is not assigned to this PR",
	ErrNoCandidate:   "no active replacement candidate in team",
	ErrNotFound:      "resource not found",
	ErrInternalIssue: "internal server issue, please try again",
}

// statusByCode - HTTP-статусы по коду.
var statusByCode = map[Code]int{
	ErrTeamExists:    http.StatusBadRequest,
	ErrPRExists:      http.StatusConflict,
	ErrPRMerged:      http.StatusConflict,
	ErrNotAssigned:   http.StatusConflict,
	ErrNoCandidate:   http.StatusConflict,
	ErrNotFound:      http.StatusNotFound,
	ErrInternalIssue: http.StatusInternalServerError,
}

// New создаёт AppError по коду.
func New(code Code) *AppError {
	return &AppError{Code: code, Message: messageFor(code)}
}

// FromCode возвращает сообщение по коду (без создания AppError).
func FromCode(code Code) string { return messageFor(code) }

func messageFor(code Code) string {
	if m, ok := messages[code]; ok {
		return m
	}
	return messages[ErrInternalIssue]
}
