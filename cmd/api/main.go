package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/quasttyy/pr-reviewer/internal/config"
	"github.com/quasttyy/pr-reviewer/internal/postgres"
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

	_ = pool // будет использоваться далее в репозиториях

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

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