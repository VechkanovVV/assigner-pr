// Package storage содержит модели данных и интерфейсы репозиториев.
package storage

import "time"

// PRStatus - статус Pull Request.
type PRStatus string

const (
	// StatusOpen - PR открыт и ожидает ревью.
	StatusOpen PRStatus = "OPEN"
	// StatusMerged - PR смержен.
	StatusMerged PRStatus = "MERGED"
)

// User - пользователь, участник команды.
type User struct {
	UpdatedAt time.Time
	ID        string
	Username  string
	TeamID    int
	IsActive  bool
}

// Team - команда разработчиков.
type Team struct {
	TeamName  string
	CreatedAt time.Time
	Members   []User
	ID        int
}

// PullRequest - PR с ревьюверами.
type PullRequest struct {
	ID                string
	Name              string
	AuthorID          string
	Status            PRStatus
	CreatedAt         time.Time
	MergedAt          *time.Time
	AssignedReviewers []string
}
