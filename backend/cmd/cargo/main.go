// cargo — единая команда приложения: serve, migrate, create-admin, seed, openapi.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/borablad/krgo_crm_mvp/backend/internal/app"
	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/config"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
	"github.com/borablad/krgo_crm_mvp/backend/internal/seed"
	"github.com/borablad/krgo_crm_mvp/backend/internal/users"
)

const usage = `Использование: cargo <команда>

  serve         запустить HTTP-сервер (миграции применяются при старте, если CARGO_MIGRATE_ON_START=true)
  migrate       применить миграции и выйти
  create-admin  создать администратора: cargo create-admin -login admin -name "Имя" -password ...
  seed          заполнить базу демонстрационными данными
  openapi       напечатать спецификацию OpenAPI (база не нужна)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		os.Exit(1)
	}
}

func run(cmd string, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		// Для openapi база и остальные настройки не нужны.
		if cmd != "openapi" {
			return err
		}
	}
	log := newLogger(cfg)

	switch cmd {
	case "openapi":
		spec, err := app.New(cfg, nil, log).OpenAPI()
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(append(spec, '\n'))
		return err

	case "migrate":
		return pg.Migrate(ctx, cfg.DatabaseURL)

	case "serve":
		if cfg.MigrateOnStart {
			if err := pg.Migrate(ctx, cfg.DatabaseURL); err != nil {
				return err
			}
		}
		pool, err := pg.NewPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		return app.New(cfg, pool, log).Serve(ctx)

	case "create-admin":
		fs := flag.NewFlagSet("create-admin", flag.ContinueOnError)
		login := fs.String("login", "admin", "логин")
		name := fs.String("name", "Администратор", "имя")
		password := fs.String("password", "", "пароль (не короче 8 символов)")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if *password == "" {
			return errors.New("укажите -password")
		}
		pool, err := pg.NewPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		a := app.New(cfg, pool, log)
		u, err := a.Users.Create(ctx, users.CreateInput{Login: *login, FullName: *name, Role: auth.RoleAdmin, Password: *password})
		if err != nil {
			return err
		}
		fmt.Printf("создан администратор %s (id %d)\n", u.Login, u.ID)
		return nil

	case "seed":
		pool, err := pg.NewPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		return seed.Run(ctx, app.New(cfg, pool, log), os.Stdout)

	default:
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("неизвестная команда %q", cmd)
	}
}

func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if cfg.IsProd() {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
