// Package handlers содержит HTTP-обработчики
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/VechkanovVV/assigner-pr/internal/api/dto"
	"github.com/VechkanovVV/assigner-pr/internal/service"
)

// PRHandler обёртка над service.PRService для HTTP-эндпоинтов PR.
type PRHandler struct {
	PRService *service.PRService
}

// NewPRHandler возвращает новый PRHandler.
func NewPRHandler(prService *service.PRService) *PRHandler {
	return &PRHandler{PRService: prService}
}

// CreatePR обрабатывает POST /pullRequest/create
func (p *PRHandler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePRRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, string(InvalidRequest), "invalid JSON")
		return
	}

	pr, appErr := p.PRService.CreatePR(r.Context(), req.PullRequestID, req.PullRequestName, req.AuthorID)

	if appErr != nil {
		respondAppError(w, appErr)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"pr": dto.FromStoragePR(pr),
	})
}

// Merge обрабатывает POST /pullRequest/merge
func (p *PRHandler) Merge(w http.ResponseWriter, r *http.Request) {
	var req dto.MergeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, string(InvalidRequest), "invalid JSON")
		return
	}

	pr, appErr := p.PRService.Merge(r.Context(), req.PullRequestID)

	if appErr != nil {
		respondAppError(w, appErr)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"pr": dto.FromStoragePR(pr),
	})
}

// ReassignReviewer обрабатывает POST /pullRequest/reassign.
func (p *PRHandler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req dto.ReassignRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, string(InvalidRequest), "invalid JSON")
		return
	}

	pr, replacedBy, appErr := p.PRService.ReassignReviewer(r.Context(), req.PullRequestID, req.OldReviewerID)

	if appErr != nil {
		respondAppError(w, appErr)
		return
	}

	respondJSON(w, http.StatusOK, dto.FromStoragePRWithReplacedBy(pr, replacedBy))
}
