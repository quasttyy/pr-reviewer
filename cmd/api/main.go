package main

import (
	"context"

	"github.com/quasttyy/pr-reviewer/internal/config"
	"github.com/quasttyy/pr-reviewer/internal/postgres"
	"github.com/quasttyy/pr-reviewer/internal/utils"
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
}