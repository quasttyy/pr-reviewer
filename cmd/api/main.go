package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/quasttyy/pr-reviewer/internal/config"
	"github.com/quasttyy/pr-reviewer/internal/handlers"
	"github.com/quasttyy/pr-reviewer/internal/postgres"
	"github.com/quasttyy/pr-reviewer/internal/repo"
	"github.com/quasttyy/pr-reviewer/internal/service"
	logger "github.com/quasttyy/pr-reviewer/internal/utils"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Загружаем конфиг
	cfg := config.MustLoad("config.yaml")
	
	// Инициализируем логгер 
	logger.Init(cfg.Env)

	// Инициализируем контекст
	ctx := context.Background()

	// Инициализируем пул соединений
	pool := postgres.InitPool(
		ctx,
		cfg.Database.DSN,
		cfg.Database.MinConns,
		cfg.Database.MaxConns,
	)
	defer pool.Close()

	// Инициализируем репозитории и сервисы
	teamRepo := repo.NewTeamRepo(pool)
	teamSvc := service.NewTeamService(teamRepo)
	teamH := handlers.NewTeamHandlers(teamSvc)
	userRepo := repo.NewUserRepo(pool)
	prRepo := repo.NewPRRepo(pool)
	userSvc := service.NewUserService(userRepo, prRepo)
	userH := handlers.NewUserHandlers(userSvc)
	prSvc := service.NewPRService(prRepo)
	prH := handlers.NewPRHandlers(prSvc)

	// Создаем роутер chi
	r := chi.NewRouter()

	// Health
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Team
	r.Route("/team", func(rt chi.Router) {
		rt.Post("/add", teamH.AddTeam)
		rt.Get("/get", teamH.GetTeam)
	})

	// Users
	r.Route("/users", func(ru chi.Router) {
		ru.Post("/setIsActive", userH.SetIsActive)
		ru.Get("/getReview", userH.GetReview)
	})

	// Pull Requests
	r.Route("/pullRequest", func(rp chi.Router) {
		rp.Post("/create", prH.Create)
		rp.Post("/merge", prH.Merge)
		rp.Post("/reassign", prH.Reassign)
	})

	// Указываем адрес и порт
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Запускаем сервер
	logger.Info("starting http server", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("http server failed", "error", err)
	}
}