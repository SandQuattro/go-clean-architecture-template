package config

import (
	"fmt"
	"log/slog"
	"net"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		App     `json:"app"     toml:"app"`
		HTTP    `json:"http"    toml:"http"`
		DB      `json:"db"      toml:"db"`
		Log     `json:"logger"  toml:"logger"`
		Tracing `json:"tracing" toml:"tracing"`
	}

	App struct {
		Name        string `json:"name"        toml:"name"        env:"APP_NAME"`
		Environment string `json:"environment" toml:"environment" env:"ENV_NAME" env-default:"dev"`
		Debug       bool   `json:"debug"       toml:"debug"       env:"DEBUG"    env-default:"false"`
	}

	HTTP struct {
		Port string ` json:"port" toml:"port" env:"HTTP_PORT"`
	}

	DB struct {
		DBHost     string `json:"host"     toml:"host"     env:"DB_HOST"`
		DBPort     int    `json:"port"     toml:"port"     env:"DB_PORT"`
		DBUser     string `json:"user"     toml:"user"     env:"DB_USER"`
		DBPassword string `json:"password" toml:"password" env:"DB_PASSWORD" env-required:"true"`
		DBName     string `json:"name"     toml:"name"     env:"DB_NAME"`
		PoolMax    int32  `json:"pool_max" toml:"pool_max" env:"PG_POOL_MAX" env-required:"true"`
	}

	Log struct {
		Level slog.Level `json:"level" toml:"level" env:"LOG_LEVEL"`
	}

	Tracing struct {
		URL string ` json:"url" toml:"url" env:"TRACING_URL"`
	}
)

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)

	configTomlPath := filepath.Join(basePath, "config.toml")
	err := cleanenv.ReadConfig(configTomlPath, cfg)
	if err != nil {
		configJsonPath := filepath.Join(basePath, "config.json")
		err = cleanenv.ReadConfig(configJsonPath, cfg)
		if err != nil {
			return nil, fmt.Errorf("config error: %w", err)
		}
	}

	err = cleanenv.ReadEnv(cfg)
	if err != nil {
		return nil, fmt.Errorf("env read error: %w", err)
	}

	if strings.Contains(cfg.DBHost, ":") {
		host, port, e := net.SplitHostPort(cfg.DBHost)
		if e != nil {
			return nil, e
		}
		cfg.DBHost = host
		cfg.DBPort, err = strconv.Atoi(port)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
