// Package service содержит бизнес-логику приложения.
package service

import (
	"context"

	"github.com/VechkanovVV/assigner-pr/internal/apperrors"
	"github.com/VechkanovVV/assigner-pr/internal/storage"
)

// TeamService - сервис для управления командами.
type TeamService struct {
	teamRepo storage.TeamRepository
}

// NewTeamService возвращает новый TeamService.
func NewTeamService(teamRepo storage.TeamRepository) *TeamService {
	return &TeamService{teamRepo: teamRepo}
}

// CreateTeam создаёт новую команду.
func (t *TeamService) CreateTeam(ctx context.Context, team storage.Team) *apperrors.AppError {
	return t.teamRepo.Create(ctx, team)
}

// GetTeamByName возвращает команду по имени.
func (t *TeamService) GetTeamByName(ctx context.Context, teamName string) (storage.Team, *apperrors.AppError) {
	return t.teamRepo.GetByName(ctx, teamName)
}

// GetTeamByID возвращает команду по id.
func (t *TeamService) GetTeamByID(ctx context.Context, teamID int) (storage.Team, *apperrors.AppError) {
	return t.teamRepo.GetByID(ctx, teamID)
}
