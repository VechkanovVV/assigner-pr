package handlers

import (
	"net/http"

	"github.com/VechkanovVV/assigner-pr/internal/service"
)

// StatsHandler обрабатывает запросы статистики.
type StatsHandler struct {
	PRService *service.PRService
}

// NewStatsHandler создаёт новый StatsHandler.
func NewStatsHandler(prService *service.PRService) *StatsHandler {
	return &StatsHandler{PRService: prService}
}

// GetAssignments - GET /stats/assignments
func (s *StatsHandler) GetAssignments(w http.ResponseWriter, r *http.Request) {
	byUser, byPR, appErr := s.PRService.GetAssignmentStats(r.Context())
	if appErr != nil {
		respondAppError(w, appErr)
		return
	}

	resp := map[string]any{
		"assignments_by_user": byUser,
		"assignments_by_pr":   byPR,
	}
	respondJSON(w, http.StatusOK, resp)
}
