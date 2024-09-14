package config

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		App     `json:"app"`
		HTTP    `json:"http"`
		DB      `json:"db"`
		Log     `json:"logger"`
		Tracing `json:"tracing"`
	}

	App struct {
		Name    string `env-required:"false" json:"name"    env:"APP_NAME"`
		Version string `env-required:"false" json:"version" env:"APP_VERSION"`
	}

	HTTP struct {
		Port string `env-required:"false" json:"port" env:"HTTP_PORT"`
	}

	DB struct {
		DBHost     string `env-required:"false" json:"host"     env:"DB_HOST"`
		DBPort     int    `env-required:"false" json:"port"     env:"DB_PORT"`
		DBUser     string `env-required:"false" json:"user"     env:"DB_USER"`
		DBPassword string `env-required:"true"  json:"password" env:"DB_PASSWORD"`
		DBName     string `env-required:"false" json:"name"     env:"DB_NAME"`
		PoolMax    int32  `env-required:"true"  json:"pool_max" env:"PG_POOL_MAX"`
	}

	Log struct {
		Level slog.Level `env-required:"false" json:"level"   env:"LOG_LEVEL"`
	}

	Tracing struct {
		URL string `env-required:"false" json:"url" env:"TRACING_URL"`
	}
)

// LoadConfig returns app config.
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	configPath := filepath.Join(basePath, "config.json")

	err := cleanenv.ReadConfig(configPath, cfg)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	err = cleanenv.ReadEnv(cfg)
	if err != nil {
		return nil, fmt.Errorf("env read error: %w", err)
	}

	return cfg, nil
}
