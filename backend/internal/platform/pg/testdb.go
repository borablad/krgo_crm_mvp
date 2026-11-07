package pg

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPool подключается к тестовой базе из TEST_DATABASE_URL, применяет миграции
// и очищает все таблицы. Без переменной тест пропускается, чтобы `go test` работал и без Postgres.
func TestPool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL не задан, тест с базой пропущен")
	}
	ctx := context.Background()
	if err := Migrate(ctx, url); err != nil {
		t.Fatalf("миграции тестовой базы: %v", err)
	}
	pool, err := NewPool(ctx, url)
	if err != nil {
		t.Fatalf("подключение к тестовой базе: %v", err)
	}
	t.Cleanup(pool.Close)
	truncateAll(t, pool)
	return pool
}

// truncateAll чистит все таблицы схемы public, кроме служебной таблицы goose
// и справочников, заполняемых миграциями (единицы измерения, валюты).
func truncateAll(t testing.TB, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	rows, err := pool.Query(ctx, `select tablename from pg_tables where schemaname = 'public' and tablename not in ('goose_db_version', 'units', 'currencies')`)
	if err != nil {
		t.Fatalf("список таблиц: %v", err)
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		names = append(names, `"`+n+`"`)
	}
	rows.Close()
	if len(names) == 0 {
		return
	}
	if _, err := pool.Exec(ctx, "truncate "+strings.Join(names, ", ")+" restart identity cascade"); err != nil {
		t.Fatalf("очистка таблиц: %v", err)
	}
}
