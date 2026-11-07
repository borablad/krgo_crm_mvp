#!/usr/bin/env bash
# Восстановление базы из копии: ./restore.sh backups/cargo-2026-11-07_0230.sql.gz
# Останавливает приложение, пересоздаёт базу, загружает дамп, запускает приложение.
set -euo pipefail
cd "$(dirname "$0")"
dump="${1:?укажите файл копии}"
[ -f "$dump" ] || { echo "нет файла $dump"; exit 1; }
docker compose stop app
docker compose exec -T db psql -U cargo -d postgres -v ON_ERROR_STOP=1 \
  -c "drop database if exists cargo with (force)" -c "create database cargo owner cargo"
gunzip -c "$dump" | docker compose exec -T db psql -U cargo -d cargo -v ON_ERROR_STOP=1 -q
docker compose start app
echo "восстановлено из $dump"
