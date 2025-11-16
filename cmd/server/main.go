// Package main - точка входа в приложение.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VechkanovVV/assigner-pr/internal/api/handlers"
	"github.com/VechkanovVV/assigner-pr/internal/api/router"
	"github.com/VechkanovVV/assigner-pr/internal/config"
	"github.com/VechkanovVV/assigner-pr/internal/infra/postgres"
	"github.com/VechkanovVV/assigner-pr/internal/service"
	postgresRepo "github.com/VechkanovVV/assigner-pr/internal/storage/postgres"
)

func main() {
	ctx := context.Background()

	dbCfg := config.LoadDB()
	log.Printf("starting server with DB config: host=%s port=%d dbname=%s sslmode=%s",
		dbCfg.Host, dbCfg.Port, dbCfg.Name, dbCfg.SSLmode)

	pool, err := postgres.NewPool(
		ctx,
		dbCfg.Port,
		dbCfg.Host,
		dbCfg.User,
		dbCfg.Password,
		dbCfg.Name,
		string(dbCfg.SSLmode),
	)
	if err != nil {
		log.Fatalf("failed to create DB pool: %v", err)
	}
	log.Println("database connection pool created successfully")

	teamRepo := postgresRepo.NewTeamRepository(pool)
	userRepo := postgresRepo.NewUserRepository(pool)
	prRepo := postgresRepo.NewPullRequestRepository(pool)

	teamService := service.NewTeamService(teamRepo)
	userService := service.NewUserService(userRepo, prRepo)
	prService := service.NewPRService(userRepo, prRepo)

	teamHandler := handlers.NewTeamHandler(teamService)
	userHandler := handlers.NewUserHandler(userService, teamService)
	prHandler := handlers.NewPRHandler(prService)

	handler := router.NewRouter(teamHandler, userHandler, prHandler)

	serverCfg := config.LoadServer()
	srv := &http.Server{
		Addr:         serverCfg.Addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("starting HTTP server on %s", serverCfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-quit
	log.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	if err := srv.Shutdown(shutdownCtx); err != nil {
		cancel()
		pool.Close()
		log.Fatalf("server forced to shutdown: %v", err)
	}

	cancel()
	pool.Close()
	log.Println("server exited gracefully")
}
