package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"

	"github.com/VechkanovVV/assigner-pr/internal/api/dto"
	"github.com/VechkanovVV/assigner-pr/internal/infra/postgres"
)

type APIIntegrationTestSuite struct {
	suite.Suite
	httpClient *http.Client
	dbPool     *pgxpool.Pool
	baseURL    string
}

func TestAPIIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(APIIntegrationTestSuite))
}

func (s *APIIntegrationTestSuite) SetupSuite() {
	s.baseURL = "http://localhost:8080"
	s.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	dbHost := getenv("INTEGRATION_DB_HOST", "localhost")
	dbPortStr := getenv("INTEGRATION_DB_PORT", "5432")
	dbUser := getenv("INTEGRATION_DB_USER", "admin")
	dbPassword := getenv("INTEGRATION_DB_PASSWORD", "admin")
	dbName := getenv("INTEGRATION_DB_NAME", "db")
	dbSSLMode := getenv("INTEGRATION_DB_SSLMODE", "disable")

	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		log.Fatalf("Invalid INTEGRATION_DB_PORT value: %v", err)
	}

	s.waitForServiceReady()

	ctx := context.Background()
	pool, err := postgres.NewPool(
		ctx,
		dbPort,
		dbHost,
		dbUser,
		dbPassword,
		dbName,
		dbSSLMode,
	)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}
	s.dbPool = pool
	s.cleanDatabase()
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func (s *APIIntegrationTestSuite) TearDownSuite() {
	if s.dbPool != nil {
		s.dbPool.Close()
	}
}

func (s *APIIntegrationTestSuite) SetupTest() {
	s.cleanDatabase()
}

func (s *APIIntegrationTestSuite) waitForServiceReady() {
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		resp, err := http.Get(s.baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			fmt.Println("Service is ready!")
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		fmt.Printf("Waiting for service to be ready... (attempt %d/%d)\n", i+1, maxAttempts)
		time.Sleep(2 * time.Second)
	}
	log.Fatal("Service did not become ready in time")
}

func (s *APIIntegrationTestSuite) cleanDatabase() {
	ctx := context.Background()
	queries := []string{
		"DELETE FROM reviews",
		"DELETE FROM pull_requests",
		"DELETE FROM users",
		"DELETE FROM teams",
	}

	for _, query := range queries {
		_, err := s.dbPool.Exec(ctx, query)
		if err != nil {
			log.Printf("Failed to clean table: %v", err)
		}
	}
}

func (s *APIIntegrationTestSuite) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var jsonBody []byte
	var err error

	if body != nil {
		jsonBody, err = json.Marshal(body)
		s.Require().NoError(err)
	}

	req, err := http.NewRequest(method, s.baseURL+endpoint, bytes.NewBuffer(jsonBody))
	s.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")
	return s.httpClient.Do(req)
}

func (s *APIIntegrationTestSuite) TestCreateAndGetTeam() {
	teamReq := dto.TeamRequest{
		TeamName: "backend-team",
		Members: []dto.TeamMember{
			{UserID: "user1", Username: "Alice", IsActive: true},
			{UserID: "user2", Username: "Bob", IsActive: true},
			{UserID: "user3", Username: "Charlie", IsActive: true},
		},
	}

	resp, err := s.makeRequest("POST", "/team/add", teamReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)

	var createResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResp)
	resp.Body.Close()
	s.Require().NoError(err)

	resp, err = s.makeRequest("GET", "/team/get?team_name=backend-team", nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var teamResp dto.TeamResponse
	err = json.NewDecoder(resp.Body).Decode(&teamResp)
	resp.Body.Close()
	s.Require().NoError(err)

	s.Assert().Equal("backend-team", teamResp.TeamName)
	s.Assert().Len(teamResp.Members, 3)
}

func (s *APIIntegrationTestSuite) TestCreateDuplicateTeam() {
	teamReq := dto.TeamRequest{
		TeamName: "duplicate-team",
		Members: []dto.TeamMember{
			{UserID: "user1", Username: "Alice", IsActive: true},
		},
	}

	resp, err := s.makeRequest("POST", "/team/add", teamReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	resp, err = s.makeRequest("POST", "/team/add", teamReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode)

	var errorResp dto.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	resp.Body.Close()
	s.Require().NoError(err)

	s.Assert().Equal("TEAM_EXISTS", errorResp.Error.Code)
}

func (s *APIIntegrationTestSuite) TestGetNonExistentTeam() {
	resp, err := s.makeRequest("GET", "/team/get?team_name=nonexistent", nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)

	var errorResp dto.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	resp.Body.Close()
	s.Require().NoError(err)

	s.Assert().Equal("NOT_FOUND", errorResp.Error.Code)
}

func (s *APIIntegrationTestSuite) TestSetUserActiveStatus() {
	teamReq := dto.TeamRequest{
		TeamName: "user-test-team",
		Members: []dto.TeamMember{
			{UserID: "user1", Username: "Alice", IsActive: true},
			{UserID: "user2", Username: "Bob", IsActive: true},
		},
	}

	resp, err := s.makeRequest("POST", "/team/add", teamReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	setActiveReq := dto.SetActiveRequest{
		UserID:   "user1",
		IsActive: false,
	}

	resp, err = s.makeRequest("POST", "/users/setIsActive", setActiveReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var userResp map[string]dto.UserDetail
	err = json.NewDecoder(resp.Body).Decode(&userResp)
	resp.Body.Close()
	s.Require().NoError(err)

	s.Assert().Equal("user1", userResp["user"].UserID)
	s.Assert().False(userResp["user"].IsActive)
	s.Assert().Equal("user-test-team", userResp["user"].TeamName)
}

func (s *APIIntegrationTestSuite) TestSetActiveStatusForNonExistentUser() {
	setActiveReq := dto.SetActiveRequest{
		UserID:   "nonexistent-user",
		IsActive: false,
	}

	resp, err := s.makeRequest("POST", "/users/setIsActive", setActiveReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusNotFound, resp.StatusCode)

	var errorResp dto.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	resp.Body.Close()
	s.Require().NoError(err)

	s.Assert().Equal("NOT_FOUND", errorResp.Error.Code)
}

func (s *APIIntegrationTestSuite) TestCreatePRWithReviewers() {
	teamReq := dto.TeamRequest{
		TeamName: "pr-test-team",
		Members: []dto.TeamMember{
			{UserID: "author1", Username: "Author", IsActive: true},
			{UserID: "reviewer1", Username: "Reviewer1", IsActive: true},
			{UserID: "reviewer2", Username: "Reviewer2", IsActive: true},
			{UserID: "reviewer3", Username: "Reviewer3", IsActive: true},
		},
	}

	resp, err := s.makeRequest("POST", "/team/add", teamReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	prReq := dto.CreatePRRequest{
		PullRequestID:   "pr-001",
		PullRequestName: "Test Feature",
		AuthorID:        "author1",
	}

	resp, err = s.makeRequest("POST", "/pullRequest/create", prReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)

	var prResp map[string]dto.PullRequestResponse
	err = json.NewDecoder(resp.Body).Decode(&prResp)
	resp.Body.Close()
	s.Require().NoError(err)

	pr := prResp["pr"]
	s.Assert().Equal("pr-001", pr.PullRequestID)
	s.Assert().Equal("Test Feature", pr.PullRequestName)
	s.Assert().Equal("author1", pr.AuthorID)
	s.Assert().Equal("OPEN", pr.Status)
	s.Assert().Len(pr.AssignedReviewers, 2)
}

func (s *APIIntegrationTestSuite) TestCreatePRWithInactiveTeamMembers() {
	teamReq := dto.TeamRequest{
		TeamName: "inactive-team",
		Members: []dto.TeamMember{
			{UserID: "author1", Username: "Author", IsActive: true},
			{UserID: "reviewer1", Username: "Reviewer1", IsActive: false},
			{UserID: "reviewer2", Username: "Reviewer2", IsActive: false},
		},
	}

	resp, err := s.makeRequest("POST", "/team/add", teamReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	prReq := dto.CreatePRRequest{
		PullRequestID:   "pr-002",
		PullRequestName: "Test Feature",
		AuthorID:        "author1",
	}

	resp, err = s.makeRequest("POST", "/pullRequest/create", prReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)

	var prResp map[string]dto.PullRequestResponse
	err = json.NewDecoder(resp.Body).Decode(&prResp)
	resp.Body.Close()
	s.Require().NoError(err)

	pr := prResp["pr"]
	s.Assert().Equal("pr-002", pr.PullRequestID)
	s.Assert().Len(pr.AssignedReviewers, 0)
}

func (s *APIIntegrationTestSuite) TestCreateDuplicatePR() {
	teamReq := dto.TeamRequest{
		TeamName: "duplicate-pr-team",
		Members: []dto.TeamMember{
			{UserID: "author1", Username: "Author", IsActive: true},
			{UserID: "reviewer1", Username: "Reviewer", IsActive: true},
		},
	}

	resp, err := s.makeRequest("POST", "/team/add", teamReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	prReq := dto.CreatePRRequest{
		PullRequestID:   "duplicate-pr",
		PullRequestName: "Test Feature",
		AuthorID:        "author1",
	}

	resp, err = s.makeRequest("POST", "/pullRequest/create", prReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	resp, err = s.makeRequest("POST", "/pullRequest/create", prReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode)

	var errorResp dto.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	resp.Body.Close()
	s.Require().NoError(err)

	s.Assert().Equal("PR_EXISTS", errorResp.Error.Code)
}

func (s *APIIntegrationTestSuite) TestMergePR() {
	teamReq := dto.TeamRequest{
		TeamName: "merge-test-team",
		Members: []dto.TeamMember{
			{UserID: "author1", Username: "Author", IsActive: true},
			{UserID: "reviewer1", Username: "Reviewer", IsActive: true},
		},
	}

	resp, err := s.makeRequest("POST", "/team/add", teamReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	prReq := dto.CreatePRRequest{
		PullRequestID:   "pr-to-merge",
		PullRequestName: "Feature to merge",
		AuthorID:        "author1",
	}

	resp, err = s.makeRequest("POST", "/pullRequest/create", prReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	mergeReq := dto.MergeRequest{
		PullRequestID: "pr-to-merge",
	}

	resp, err = s.makeRequest("POST", "/pullRequest/merge", mergeReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var mergeResp map[string]dto.PullRequestResponse
	err = json.NewDecoder(resp.Body).Decode(&mergeResp)
	resp.Body.Close()
	s.Require().NoError(err)

	pr := mergeResp["pr"]
	s.Assert().Equal("MERGED", pr.Status)
	s.Assert().NotNil(pr.MergedAt)
}

func (s *APIIntegrationTestSuite) TestReassignReviewer() {
	teamReq := dto.TeamRequest{
		TeamName: "reassign-test-team",
		Members: []dto.TeamMember{
			{UserID: "author1", Username: "Author", IsActive: true},
			{UserID: "reviewer1", Username: "Reviewer1", IsActive: true},
			{UserID: "reviewer2", Username: "Reviewer2", IsActive: true},
			{UserID: "reviewer3", Username: "Reviewer3", IsActive: true},
		},
	}

	resp, err := s.makeRequest("POST", "/team/add", teamReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	prReq := dto.CreatePRRequest{
		PullRequestID:   "pr-reassign",
		PullRequestName: "Reassign Test",
		AuthorID:        "author1",
	}

	resp, err = s.makeRequest("POST", "/pullRequest/create", prReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)

	var prResp map[string]dto.PullRequestResponse
	err = json.NewDecoder(resp.Body).Decode(&prResp)
	resp.Body.Close()
	s.Require().NoError(err)

	originalPR := prResp["pr"]
	s.Require().Len(originalPR.AssignedReviewers, 2)

	reassignReq := dto.ReassignRequest{
		PullRequestID: "pr-reassign",
		OldReviewerID: originalPR.AssignedReviewers[0],
	}

	resp, err = s.makeRequest("POST", "/pullRequest/reassign", reassignReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var reassignResp dto.ReassignResponse
	err = json.NewDecoder(resp.Body).Decode(&reassignResp)
	resp.Body.Close()
	s.Require().NoError(err)

	s.Assert().NotEqual(originalPR.AssignedReviewers[0], reassignResp.ReplacedBy)
	s.Assert().Len(reassignResp.PullRequest.AssignedReviewers, 2)
}

func (s *APIIntegrationTestSuite) TestReassignReviewerOnMergedPR() {
	teamReq := dto.TeamRequest{
		TeamName: "merged-reassign-team",
		Members: []dto.TeamMember{
			{UserID: "author1", Username: "Author", IsActive: true},
			{UserID: "reviewer1", Username: "Reviewer1", IsActive: true},
		},
	}

	resp, err := s.makeRequest("POST", "/team/add", teamReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	prReq := dto.CreatePRRequest{
		PullRequestID:   "merged-pr",
		PullRequestName: "Merged Feature",
		AuthorID:        "author1",
	}

	resp, err = s.makeRequest("POST", "/pullRequest/create", prReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	mergeReq := dto.MergeRequest{PullRequestID: "merged-pr"}
	resp, err = s.makeRequest("POST", "/pullRequest/merge", mergeReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	reassignReq := dto.ReassignRequest{
		PullRequestID: "merged-pr",
		OldReviewerID: "reviewer1",
	}

	resp, err = s.makeRequest("POST", "/pullRequest/reassign", reassignReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusConflict, resp.StatusCode)

	var errorResp dto.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	resp.Body.Close()
	s.Require().NoError(err)

	s.Assert().Equal("PR_MERGED", errorResp.Error.Code)
}

func (s *APIIntegrationTestSuite) TestGetUserReviews() {
	teamReq := dto.TeamRequest{
		TeamName: "reviews-test-team",
		Members: []dto.TeamMember{
			{UserID: "author1", Username: "Author", IsActive: true},
			{UserID: "reviewer1", Username: "Reviewer1", IsActive: true},
			{UserID: "reviewer2", Username: "Reviewer2", IsActive: true},
		},
	}

	resp, err := s.makeRequest("POST", "/team/add", teamReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	prReq := dto.CreatePRRequest{
		PullRequestID:   "pr-for-reviews",
		PullRequestName: "Review Test",
		AuthorID:        "author1",
	}

	resp, err = s.makeRequest("POST", "/pullRequest/create", prReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	resp, err = s.makeRequest("GET", "/users/getReview?user_id=reviewer1", nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var reviewsResp dto.UserReviewsResponse
	err = json.NewDecoder(resp.Body).Decode(&reviewsResp)
	resp.Body.Close()
	s.Require().NoError(err)

	s.Assert().Equal("reviewer1", reviewsResp.UserID)
	s.Assert().Len(reviewsResp.PullRequests, 1)
	s.Assert().Equal("pr-for-reviews", reviewsResp.PullRequests[0].PullRequestID)
}

func (s *APIIntegrationTestSuite) TestCompleteWorkflow() {
	teamReq := dto.TeamRequest{
		TeamName: "workflow-team",
		Members: []dto.TeamMember{
			{UserID: "dev1", Username: "Developer1", IsActive: true},
			{UserID: "dev2", Username: "Developer2", IsActive: true},
			{UserID: "dev3", Username: "Developer3", IsActive: true},
			{UserID: "dev4", Username: "Developer4", IsActive: true},
		},
	}

	resp, err := s.makeRequest("POST", "/team/add", teamReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	prReq := dto.CreatePRRequest{
		PullRequestID:   "workflow-pr",
		PullRequestName: "Complete Workflow Feature",
		AuthorID:        "dev1",
	}

	resp, err = s.makeRequest("POST", "/pullRequest/create", prReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)

	var prResp map[string]dto.PullRequestResponse
	err = json.NewDecoder(resp.Body).Decode(&prResp)
	resp.Body.Close()
	s.Require().NoError(err)

	originalPR := prResp["pr"]
	s.Require().Len(originalPR.AssignedReviewers, 2)

	reassignReq := dto.ReassignRequest{
		PullRequestID: "workflow-pr",
		OldReviewerID: originalPR.AssignedReviewers[0],
	}

	resp, err = s.makeRequest("POST", "/pullRequest/reassign", reassignReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()
	setActiveReq := dto.SetActiveRequest{
		UserID:   "dev2",
		IsActive: false,
	}

	resp, err = s.makeRequest("POST", "/users/setIsActive", setActiveReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	mergeReq := dto.MergeRequest{PullRequestID: "workflow-pr"}
	resp, err = s.makeRequest("POST", "/pullRequest/merge", mergeReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var mergeResp map[string]dto.PullRequestResponse
	err = json.NewDecoder(resp.Body).Decode(&mergeResp)
	resp.Body.Close()
	s.Require().NoError(err)

	s.Assert().Equal("MERGED", mergeResp["pr"].Status)

	resp, err = s.makeRequest("GET", "/team/get?team_name=workflow-team", nil)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var teamResp dto.TeamResponse
	err = json.NewDecoder(resp.Body).Decode(&teamResp)
	resp.Body.Close()
	s.Require().NoError(err)

	s.Assert().Len(teamResp.Members, 4)

	var dev2Found bool
	for _, member := range teamResp.Members {
		if member.UserID == "dev2" {
			dev2Found = true
			s.Assert().False(member.IsActive)
			break
		}
	}
	s.Assert().True(dev2Found, "dev2 should be found in team members")
}
