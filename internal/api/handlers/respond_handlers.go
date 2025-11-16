package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/VechkanovVV/assigner-pr/internal/api/dto"
	"github.com/VechkanovVV/assigner-pr/internal/apperrors"
)

// InvalidType - тип ошибок запроса.
type InvalidType string

// InvalidRequest - некорректный запрос.
const InvalidRequest InvalidType = "INVALID_REQUEST"

// respondJSON отправляет JSON-ответ с заданным статусом.
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}

// respondError отправляет ошибку в формате OpenAPI ErrorResponse.
func respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:    code,
			Message: message,
		},
	}); err != nil {
		log.Printf("failed to encode error response: %v", err)
	}
}

// respondAppError маппит *apperrors.AppError в HTTP-ответ.
func respondAppError(w http.ResponseWriter, err *apperrors.AppError) {
	status := err.HTTPStatus()
	respondError(w, status, string(err.Code), err.Message)
}
