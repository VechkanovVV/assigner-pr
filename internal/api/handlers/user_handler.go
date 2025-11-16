package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/VechkanovVV/assigner-pr/internal/api/dto"
	"github.com/VechkanovVV/assigner-pr/internal/service"
)

// UserHandler обрабатывает HTTP-запросы, связанные с пользователями.
type UserHandler struct {
	UserService *service.UserService
	TeamService *service.TeamService
}

// NewUserHandler возвращает новый UserHandler.
func NewUserHandler(userService *service.UserService, teamService *service.TeamService) *UserHandler {
	return &UserHandler{
		UserService: userService,
		TeamService: teamService,
	}
}

// SetActiveStatus устанавливает флаг активности пользователя.
func (u *UserHandler) SetActiveStatus(w http.ResponseWriter, r *http.Request) {
	var req dto.SetActiveRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, string(InvalidRequest), "invalid JSON")
		return
	}

	if req.UserID == "" {
		respondError(w, http.StatusBadRequest, string(InvalidRequest), "invalid JSON")
		return
	}

	user, appErr := u.UserService.SetActiveStatus(r.Context(), req.UserID, req.IsActive)
	if appErr != nil {
		respondAppError(w, appErr)
		return
	}

	team, appErr := u.TeamService.GetTeamByID(r.Context(), user.TeamID)
	if appErr != nil {
		respondAppError(w, appErr)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"user": dto.UserDetail{
			UserID:   user.ID,
			Username: user.Username,
			TeamName: team.TeamName,
			IsActive: user.IsActive,
		},
	})
}

// GetUserReviews - GET /users/getReview.
func (u *UserHandler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")

	if userID == "" {
		respondError(w, http.StatusBadRequest, string(InvalidRequest), "user_id query parameter is required")
		return
	}

	prs, appErr := u.UserService.GetUserReviews(r.Context(), userID)
	if appErr != nil {
		respondAppError(w, appErr)
		return
	}

	respondJSON(w, http.StatusOK, dto.UserReviewsResponse{
		UserID:       userID,
		PullRequests: dto.FromStoragePRList(prs),
	})
}
