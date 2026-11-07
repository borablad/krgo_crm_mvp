// Package app собирает приложение: база → сервисы → модули → HTTP.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/borablad/krgo_crm_mvp/backend/internal/audit"
	"github.com/borablad/krgo_crm_mvp/backend/internal/auth"
	"github.com/borablad/krgo_crm_mvp/backend/internal/config"
	"github.com/borablad/krgo_crm_mvp/backend/internal/crm"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/httpx"
	"github.com/borablad/krgo_crm_mvp/backend/internal/users"
	"github.com/borablad/krgo_crm_mvp/backend/internal/web"
)

// Version подставляется при сборке через -ldflags.
var Version = "dev"

// App — собранное приложение.
type App struct {
	Config config.Config
	Pool   *pgxpool.Pool
	Log    *slog.Logger
	API    huma.API

	Auth  *auth.Service
	Users *users.Service
	Audit *audit.Service
	CRM   *crm.Service

	handler http.Handler
}

// New собирает все модули. Pool может быть nil, когда нужен только OpenAPI (команда `cargo openapi`).
func New(cfg config.Config, pool *pgxpool.Pool, log *slog.Logger) *App {
	mux := http.NewServeMux()
	api := httpx.NewAPI(mux, Version)

	recorder := audit.NewRecorder()
	authSvc := auth.NewService(pool, cfg.SessionTTL, cfg.SecureCookies)
	usersSvc := users.NewService(pool, recorder, authSvc)
	auditSvc := audit.NewService(pool)
	crmSvc := crm.NewService(pool, recorder)

	api.UseMiddleware(auth.Middleware(api, authSvc))

	auth.Register(api, authSvc)
	users.Register(api, usersSvc)
	audit.Register(api, auditSvc)
	crm.Register(api, crmSvc)
	registerHealth(api, pool)

	// Неизвестные пути API — 404 в формате API, а не index.html.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteProblem(w, http.StatusNotFound, "Такого метода API нет")
	})
	mux.Handle("/", web.Handler())

	handler := httpx.Chain(mux,
		httpx.RequestID,
		httpx.ClientIP(cfg.IsProd()),
		httpx.Logging(log),
		httpx.Recover(log),
		httpx.CSRF,
	)

	return &App{
		Config: cfg, Pool: pool, Log: log, API: api,
		Auth: authSvc, Users: usersSvc, Audit: auditSvc, CRM: crmSvc,
		handler: handler,
	}
}

// Handler — корневой HTTP-обработчик.
func (a *App) Handler() http.Handler { return a.handler }

// OpenAPI возвращает спецификацию в JSON.
func (a *App) OpenAPI() ([]byte, error) {
	return json.MarshalIndent(a.API.OpenAPI(), "", "  ")
}

// Serve запускает HTTP-сервер и корректно останавливает его по отмене контекста.
func (a *App) Serve(ctx context.Context) error {
	srv := &http.Server{
		Addr:              a.Config.Addr,
		Handler:           a.handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		a.Log.Info("сервер запущен", "addr", a.Config.Addr, "version", Version, "env", a.Config.Env)
		errCh <- srv.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("http-сервер: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type healthOutput struct {
	Body HealthResponse
}

func registerHealth(api huma.API, pool *pgxpool.Pool) {
	huma.Register(api, httpx.Public(huma.Operation{
		OperationID: "health", Method: http.MethodGet, Path: "/api/v1/health",
		Summary: "Проверка работоспособности", Tags: []string{"system"}, Security: []map[string][]string{},
	}), func(ctx context.Context, _ *struct{}) (*healthOutput, error) {
		if pool != nil {
			if err := pool.Ping(ctx); err != nil {
				return nil, huma.Error503ServiceUnavailable("База данных недоступна")
			}
		}
		out := &healthOutput{}
		out.Body.Status = "ok"
		out.Body.Version = Version
		return out, nil
	})
}
