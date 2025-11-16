package logger

import (
	"log/slog"
	"os"
)

// log — глобальный экземпляр slog.Logger
var log *slog.Logger

// Init создаёт глобальный логгер с режимом dev/prod
func Init(env string) {
	var handler slog.Handler

	switch env {
	case "prod":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	default:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	log = slog.New(handler)
	log.Info("logger initialized", slog.String("env", env))
}

// Алиасы 
func Info(msg string, kv ...any)  { log.Info(msg, kv...) }
func Warn(msg string, kv ...any)  { log.Warn(msg, kv...) }
func Error(msg string, kv ...any) { log.Error(msg, kv...) }
func Debug(msg string, kv ...any) { log.Debug(msg, kv...) }

func Fatal(msg string, kv ...any) {
	log.Error(msg, kv...)
	os.Exit(1)
}

// Base возвращает базовый *slog.Logger
func Base() *slog.Logger {
	return log
}
