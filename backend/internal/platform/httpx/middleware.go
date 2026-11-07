// Package httpx — HTTP-сервер, общие middleware и настройка Huma.
package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	clientIPKey
)

// CSRFHeader — служебный заголовок, который клиент обязан слать на любые изменяющие запросы.
// Сторонний сайт не может поставить произвольный заголовок, поэтому cookie-сессия защищена.
const CSRFHeader = "X-Requested-With"
const CSRFValue = "cargo"

// RequestID присваивает запросу идентификатор и возвращает его в заголовке.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			var b [8]byte
			_, _ = rand.Read(b[:])
			id = hex.EncodeToString(b[:])
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ClientIP кладёт в контекст адрес клиента. За Caddy настоящий адрес приходит в X-Forwarded-For.
func ClientIP(trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			if trustProxy {
				if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
					ip = strings.TrimSpace(strings.Split(fwd, ",")[0])
				}
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey, ip)))
		})
	}
}

func RequestIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

func ClientIPFrom(ctx context.Context) string {
	v, _ := ctx.Value(clientIPKey).(string)
	return v
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Logging пишет одну строку лога на запрос.
func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r)
			if sw.status == 0 {
				sw.status = http.StatusOK
			}
			level := slog.LevelInfo
			if sw.status >= 500 {
				level = slog.LevelError
			}
			log.Log(r.Context(), level, "http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"ms", time.Since(start).Milliseconds(),
				"ip", ClientIPFrom(r.Context()),
				"request_id", RequestIDFrom(r.Context()),
			)
		})
	}
}

// Recover превращает панику в ответ 500, не роняя процесс.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					log.Error("паника в обработчике", "panic", p, "path", r.URL.Path, "request_id", RequestIDFrom(r.Context()))
					WriteProblem(w, http.StatusInternalServerError, "Внутренняя ошибка сервера. Попробуйте ещё раз, если не поможет — сообщите администратору.")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CSRF требует служебный заголовок на изменяющих запросах к API.
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if strings.HasPrefix(r.URL.Path, "/api/") && r.Header.Get(CSRFHeader) != CSRFValue {
				WriteProblem(w, http.StatusForbidden, "Запрос отклонён: нет служебного заголовка "+CSRFHeader+".")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// WriteProblem пишет ошибку в том же формате RFC 7807, что и Huma.
func WriteProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": status,
		"title":  http.StatusText(status),
		"detail": detail,
	})
}

// Chain применяет middleware в порядке перечисления: первый — самый внешний.
func Chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
