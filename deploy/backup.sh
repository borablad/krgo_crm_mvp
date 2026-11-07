#!/usr/bin/env bash
# Ежедневная копия базы: /opt/cargo-crm/backups/cargo-ГГГГ-ММ-ДД_ЧЧММ.sql.gz, хранится 14 дней.
# Крон: 30 2 * * * /opt/cargo-crm/backup.sh >> /opt/cargo-crm/backups/backup.log 2>&1
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p backups
name="backups/cargo-$(date +%F_%H%M).sql.gz"
docker compose exec -T db pg_dump -U cargo -d cargo --no-owner | gzip > "$name"
find backups -name 'cargo-*.sql.gz' -mtime +14 -delete
echo "$(date -Is) ok $name $(du -h "$name" | cut -f1)"
