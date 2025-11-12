package main

import (
	"github.com/quasttyy/pr-reviewer/internal/utils"
	"github.com/quasttyy/pr-reviewer/internal/config"
)

func main() {
	// Загружаем конфиг
	cfg := config.MustLoad("config.yaml")
	
	// Инициализируем логгер 
	logger.Init(cfg.Env)
}