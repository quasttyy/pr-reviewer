package main

import (
	"os"
	"path/filepath"
	"github.com/quasttyy/pr-reviewer/internal/config"
	logger "github.com/quasttyy/pr-reviewer/internal/utils"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	// Загружаем конфиг
	cfg := config.MustLoad("config.yaml")

	// Инициализируем логгер
	logger.Init(cfg.Env)

	// Указываем путь к миграциям
	migrationsPath, err := filepath.Abs("migrations")
	if err != nil {
		logger.Fatal("failed to create migrations path:", err)
	}

	sourceURL := "file://" + migrationsPath
	dsn := cfg.Database.DSN

	logger.Info("running migrations...")
	logger.Info("migration source url", "value", sourceURL)
	logger.Info("dsn", "value", dsn)

	// Создаём мигратор
	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		logger.Fatal("failed to create migrator:", err)
	}

	// Закрываем его после завершения
	defer func() {
		_, _ = m.Close()
	}()

	// Применяем миграции
	err = m.Up()
	if err != nil {
		if err == migrate.ErrNoChange {
			logger.Info("no new migrations — DB up to date")
			os.Exit(0)
		}
		logger.Fatal("failed to run migrations:", err)
	}

	logger.Info("migrations applied successfully")
}
