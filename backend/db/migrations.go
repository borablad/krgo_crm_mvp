// Package db встраивает SQL-миграции в бинарник, чтобы контейнер мог мигрировать базу сам.
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
