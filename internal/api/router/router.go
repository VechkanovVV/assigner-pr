// Package router регистрирует HTTP-маршруты и возвращает http.Handler.
package router

import (
	"net/http"

	"github.com/VechkanovVV/assigner-pr/internal/api/handlers"
)

// NewRouter создаёт HTTP router с зарегистрированными маршрутами.
func NewRouter(
	teamHandler *handlers.TeamHandler,
	userHandler *handlers.UserHandler,
	prHandler *handlers.PRHandler,
) http.Handler {

	mux := http.NewServeMux()

	mux.HandleFunc("POST /team/add", teamHandler.CreateTeam)
	mux.HandleFunc("GET /team/get", teamHandler.GetTeam)

	mux.HandleFunc("POST /users/setIsActive", userHandler.SetActiveStatus)
	mux.HandleFunc("GET /users/getReview", userHandler.GetUserReviews)

	mux.HandleFunc("POST /pullRequest/create", prHandler.CreatePR)
	mux.HandleFunc("POST /pullRequest/merge", prHandler.Merge)
	mux.HandleFunc("POST /pullRequest/reassign", prHandler.ReassignReviewer)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)
		}
	})

	return mux
}
