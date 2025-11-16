package dto

import "github.com/VechkanovVV/assigner-pr/internal/storage"

// ToStorageTeam DTO -> storage.Team.
func (r TeamRequest) ToStorageTeam() storage.Team {
	members := make([]storage.User, 0, len(r.Members))
	for _, m := range r.Members {
		members = append(members, storage.User{
			ID:       m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}
	return storage.Team{
		TeamName: r.TeamName,
		Members:  members,
	}
}

// FromStorageTeam storage.Team -> DTO.
func FromStorageTeam(t storage.Team) TeamResponse {
	members := make([]TeamMember, 0, len(t.Members))
	for _, m := range t.Members {
		members = append(members, TeamMember{
			UserID:   m.ID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}
	return TeamResponse{
		TeamName: t.TeamName,
		Members:  members,
	}
}

// FromStoragePR storage.PullRequest -> DTO.
func FromStoragePR(pr storage.PullRequest) PullRequestResponse {
	return PullRequestResponse{
		PullRequestID:     pr.ID,
		PullRequestName:   pr.Name,
		AuthorID:          pr.AuthorID,
		Status:            string(pr.Status),
		AssignedReviewers: pr.AssignedReviewers,
		CreatedAt:         &pr.CreatedAt,
		MergedAt:          pr.MergedAt,
	}
}

// FromStoragePRShort storage.PullRequest -> короткий DTO.
func FromStoragePRShort(pr storage.PullRequest) PullRequestShort {
	return PullRequestShort{
		PullRequestID:   pr.ID,
		PullRequestName: pr.Name,
		AuthorID:        pr.AuthorID,
		Status:          string(pr.Status),
	}
}

// FromStoragePRList storage.PullRequest -> массив PullRequestShort.
func FromStoragePRList(prs []storage.PullRequest) []PullRequestShort {
	res := make([]PullRequestShort, 0, len(prs))

	for _, pr := range prs {
		res = append(res, FromStoragePRShort(pr))
	}

	return res
}

// FromStorageUser storage.User + teamName -> UserDetail.
func FromStorageUser(u storage.User, teamName string) UserDetail {
	return UserDetail{
		UserID:   u.ID,
		Username: u.Username,
		TeamName: teamName,
		IsActive: u.IsActive,
	}
}

// FromStoragePRWithReplacedBy storage.PullRequest + replaced_by -> ReassignResponse.
func FromStoragePRWithReplacedBy(pr storage.PullRequest, replacedBy string) ReassignResponse {
	return ReassignResponse{
		PullRequest: FromStoragePR(pr),
		ReplacedBy:  replacedBy,
	}
}
