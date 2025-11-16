package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Структура конфига
type Config struct {
	Env string `yaml:"env" env:"APP_ENV"`

	Server struct {
		Host string `yaml:"host" env:"SERVER_HOST"`
		Port int    `yaml:"port" env:"SERVER_PORT"`
	} `yaml:"server"`

	Database struct {
		DSN      string `yaml:"dsn" env:"DSN"`
		MaxConns int32  `yaml:"max_conns" env:"DB_MAX_CONNS"`
		MinConns int32  `yaml:"min_conns" env:"DB_MIN_CONNS"`
	} `yaml:"database"`

	Security struct {
		AdminToken string `yaml:"admin_token" env:"ADMIN_TOKEN"`
		UserToken  string `yaml:"user_token" env:"USER_TOKEN"`
	} `yaml:"security"`
}

// MustLoad читает YAML и ENV в одну структуру
func MustLoad(path string) *Config {
	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: cannot read config: %v\n", err)
		os.Exit(1)
	}
	return &cfg
}k