package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/VechkanovVV/assigner-pr/internal/api/dto"
	"github.com/VechkanovVV/assigner-pr/internal/service"
)

// TeamHandler - HTTP-запросы, связанные с командами.
type TeamHandler struct {
	TeamService *service.TeamService
}

// NewTeamHandler возвращает новый TeamHandler
func NewTeamHandler(teamService *service.TeamService) *TeamHandler {
	return &TeamHandler{TeamService: teamService}
}

// CreateTeam обрабатывает создание команды.
func (t *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req dto.TeamRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, string(InvalidRequest), "invalid JSON")
		return
	}

	if req.TeamName == "" || len(req.Members) == 0 {
		respondError(w, http.StatusBadRequest, string(InvalidRequest), "invalid JSON")
		return
	}

	team := req.ToStorageTeam()

	if appErr := t.TeamService.CreateTeam(r.Context(), team); appErr != nil {
		respondAppError(w, appErr)
		return
	}

	ct, appErr := t.TeamService.GetTeamByName(r.Context(), req.TeamName)

	if appErr != nil {
		respondAppError(w, appErr)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{
		"team": dto.FromStorageTeam(ct),
	})
}

// GetTeam поиск команды по имени.
func (t *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	tName := r.URL.Query().Get("team_name")

	if tName == "" {
		respondError(w, http.StatusBadRequest, string(InvalidRequest), "invalid JSON")
		return
	}

	team, appErr := t.TeamService.GetTeamByName(r.Context(), tName)

	if appErr != nil {
		respondAppError(w, appErr)
		return
	}
	respondJSON(w, http.StatusOK, dto.FromStorageTeam(team))
}
