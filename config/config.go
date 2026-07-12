package config

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

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
		Port string `json:"port" toml:"port" env:"HTTP_PORT" env-default:"8000"`
		// Таймауты задаются только через env: cleanenv не умеет парсить
		// time.Duration из toml/json файлов.
		ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT"     env-default:"10s"`
		WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT"    env-default:"10s"`
		IdleTimeout     time.Duration `env:"HTTP_IDLE_TIMEOUT"     env-default:"60s"`
		RequestTimeout  time.Duration `env:"HTTP_REQUEST_TIMEOUT"  env-default:"30s"`
		ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"10s"`
	}

	DB struct {
		DBHost            string `json:"host"     toml:"host"     env:"DB_HOST"`
		DBPort            int    `json:"port"     toml:"port"     env:"DB_PORT"`
		DBUser            string `json:"user"     toml:"user"     env:"DB_USER"`
		DBPassword        string `json:"password" toml:"password" env:"DB_PASSWORD" env-required:"true"`
		DBName            string `json:"name"     toml:"name"     env:"DB_NAME"`
		SSLMode           string `json:"sslmode"  toml:"sslmode"  env:"DB_SSLMODE" env-default:"disable"`
		PoolMax           int32  `json:"pool_max" toml:"pool_max" env:"PG_POOL_MAX" env-required:"true"`
		PoolMin           int32  `json:"pool_min" toml:"pool_min" env:"PG_POOL_MIN" env-default:"1"`
		ConnectTimeout    int    `json:"connect_timeout" toml:"connect_timeout" env:"PG_POOL_CONN_TIMEOUT" env-default:"5"`
		HealthCheckPeriod int    `json:"health_check_period" toml:"health_check_period" env:"PG_POOL_HEALTHCHECK" env-default:"1"`
	}

	Log struct {
		Level slog.Level `json:"level" toml:"level" env:"LOG_LEVEL"`
	}

	Tracing struct {
		URL string ` json:"url" toml:"url" env:"TRACING_URL"`
	}
)

// DSN возвращает строку подключения к Postgres; единая точка для пула и мигратора.
func (db DB) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		db.DBUser, db.DBPassword, db.DBHost, db.DBPort, db.DBName, db.SSLMode)
}

// LoadConfig ищет файл конфигурации в следующем порядке:
//  1. путь из CONFIG_PATH;
//  2. config/config.toml, config/config.json относительно рабочей директории
//     (так работают Docker-образ и запуск из корня репозитория);
//  3. рядом с исходником пакета (запуск тестов и go run из произвольной директории).
//
// Значения из env перекрывают значения из файла.
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	if err := cleanenv.ReadConfig(configPath(), cfg); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("env read error: %w", err)
	}

	if strings.Contains(cfg.DBHost, ":") {
		host, port, err := net.SplitHostPort(cfg.DBHost)
		if err != nil {
			return nil, err
		}
		cfg.DBHost = host
		cfg.DBPort, err = strconv.Atoi(port)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func configPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}

	_, sourceFile, _, _ := runtime.Caller(0)
	sourceDir := filepath.Dir(sourceFile)

	candidates := []string{
		filepath.Join("config", "config.toml"),
		filepath.Join("config", "config.json"),
		filepath.Join(sourceDir, "config.toml"),
		filepath.Join(sourceDir, "config.json"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return candidates[0]
}
