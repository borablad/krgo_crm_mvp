// Package config читает настройки приложения из переменных окружения.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Env: dev или prod. Влияет на формат логов и безопасность cookie по умолчанию.
	Env string
	// Addr — адрес HTTP-сервера, например ":8080".
	Addr string
	// DatabaseURL — строка подключения к PostgreSQL.
	DatabaseURL string
	// SecureCookies — ставить ли флаг Secure на cookie сессии. Включать, когда сайт за HTTPS.
	SecureCookies bool
	// SessionTTL — срок жизни сессии без повторного входа.
	SessionTTL time.Duration
	// MigrateOnStart — применять миграции при запуске сервера.
	MigrateOnStart bool
}

func (c Config) IsProd() bool { return c.Env == "prod" }

// Load собирает конфигурацию. Обязательна только DATABASE_URL.
func Load() (Config, error) {
	cfg := Config{
		Env:            getenv("CARGO_ENV", "dev"),
		Addr:           getenv("CARGO_ADDR", ":8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		SessionTTL:     30 * 24 * time.Hour,
		MigrateOnStart: true,
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("не задана переменная DATABASE_URL")
	}
	if cfg.Env != "dev" && cfg.Env != "prod" {
		return cfg, fmt.Errorf("CARGO_ENV должен быть dev или prod, получено %q", cfg.Env)
	}

	var err error
	if cfg.SecureCookies, err = getbool("CARGO_SECURE_COOKIES", cfg.IsProd()); err != nil {
		return cfg, err
	}
	if cfg.MigrateOnStart, err = getbool("CARGO_MIGRATE_ON_START", true); err != nil {
		return cfg, err
	}
	if v := os.Getenv("CARGO_SESSION_TTL"); v != "" {
		if cfg.SessionTTL, err = time.ParseDuration(v); err != nil {
			return cfg, fmt.Errorf("CARGO_SESSION_TTL: %w", err)
		}
	}
	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getbool(key string, def bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def, fmt.Errorf("%s: ожидалось true или false, получено %q", key, v)
	}
	return b, nil
}
