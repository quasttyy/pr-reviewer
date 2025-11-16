package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/quasttyy/pr-reviewer/internal/config"
	"github.com/quasttyy/pr-reviewer/internal/handlers"
	"github.com/quasttyy/pr-reviewer/internal/middleware"
	"github.com/quasttyy/pr-reviewer/internal/postgres"
	"github.com/quasttyy/pr-reviewer/internal/repo"
	"github.com/quasttyy/pr-reviewer/internal/service"
	logger "github.com/quasttyy/pr-reviewer/internal/utils"
)

func main() {
	// Загружаем конфиг
	cfg := config.MustLoad("config.yaml")
	
	// Инициализируем логгер 
	logger.Init(cfg.Env)

	ctx := context.Background()

	pool := postgres.InitPool(
		ctx,
		cfg.Database.DSN,
		cfg.Database.MinConns,
		cfg.Database.MaxConns,
	)
	defer pool.Close()

	teamRepo := repo.NewTeamRepo(pool)
	teamSvc := service.NewTeamService(teamRepo)
	teamH := handlers.NewTeamHandlers(teamSvc)
	userRepo := repo.NewUserRepo(pool)
	prRepo := repo.NewPRRepo(pool)
	userSvc := service.NewUserService(userRepo, prRepo)
	userH := handlers.NewUserHandlers(userSvc)
	prSvc := service.NewPRService(prRepo)
	prH := handlers.NewPRHandlers(prSvc)
	auth := middleware.NewAuth(cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// OpenAPI: /team/add без секьюрити
	mux.Handle("/team/add", http.HandlerFunc(teamH.AddTeam))
	// /team/get под UserToken или AdminToken
	mux.Handle("/team/get", auth.UserOrAdmin(http.HandlerFunc(teamH.GetTeam)))
	// users
	mux.Handle("/users/setIsActive", auth.AdminOnly(http.HandlerFunc(userH.SetIsActive)))
	mux.Handle("/users/getReview", auth.UserOrAdmin(http.HandlerFunc(userH.GetReview)))
	// pr (Admin по спекам)
	mux.Handle("/pullRequest/create", auth.AdminOnly(http.HandlerFunc(prH.Create)))
	mux.Handle("/pullRequest/merge", auth.AdminOnly(http.HandlerFunc(prH.Merge)))
	mux.Handle("/pullRequest/reassign", auth.AdminOnly(http.HandlerFunc(prH.Reassign)))

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("starting http server", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("http server failed", "error", err)
	}
}