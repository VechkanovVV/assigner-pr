// Package dto содержит структуры DTO для HTTP API.
package dto

import "time"

// ErrorResponse - формат ошибки.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail - код и сообщение об ошибке.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TeamRequest - POST /team/add body.
type TeamRequest struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

// TeamMember одержит данные команды для API.
type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

// TeamResponse - GET /team/get, POST /team/add response.
type TeamResponse struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

// UserResponse - POST /users/setIsActive response.
type UserResponse struct {
	User UserDetail `json:"user"`
}

// UserDetail содержит данные пользователя для API.
type UserDetail struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

// SetActiveRequest - POST /users/setIsActive body.
type SetActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

// CreatePRRequest - POST /pullRequest/create body.
type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

// PullRequestResponse - формат PR.
type PullRequestResponse struct {
	CreatedAt         *time.Time `json:"createdAt,omitempty"`
	MergedAt          *time.Time `json:"mergedAt,omitempty"`
	PullRequestID     string     `json:"pull_request_id"`
	PullRequestName   string     `json:"pull_request_name"`
	AuthorID          string     `json:"author_id"`
	Status            string     `json:"status"`
	AssignedReviewers []string   `json:"assigned_reviewers"`
}

// MergeRequest - POST /pullRequest/merge body.
type MergeRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

// ReassignRequest - POST /pullRequest/reassign body.
type ReassignRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldReviewerID string `json:"old_reviewer_id"`
}

// ReassignResponse - POST /pullRequest/reassign response.
type ReassignResponse struct {
	ReplacedBy  string              `json:"replaced_by"`
	PullRequest PullRequestResponse `json:"pr"`
}

// UserReviewsResponse - GET /users/getReview response.
type UserReviewsResponse struct {
	UserID       string             `json:"user_id"`
	PullRequests []PullRequestShort `json:"pull_requests"`
}

// PullRequestShort короткая версия UserReviewsResponse
type PullRequestShort struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}
