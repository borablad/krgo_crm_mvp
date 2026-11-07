# Деплой

## Стенд
- VPS: Ubuntu 22.04, 1 vCPU, 1 ГБ памяти (+2 ГБ swap), 20 ГБ диска. Docker 29, Compose v5.
- Адрес: `http://38.180.39.171` (пока без домена; с доменом Caddy включит HTTPS сам).
- Файлы стенда: `/opt/cargo-crm/` — `compose.yml`, `Caddyfile`, `.env`, `backup.sh`, `restore.sh`, папка `backups/`.
- Три контейнера: `db` (PostgreSQL 17), `app` (Go-бинарник с фронтендом внутри), `caddy` (порты 80/443).
- Сервер ничего не собирает: образ приходит из GitHub Container Registry `ghcr.io/borablad/krgo_crm_mvp`.

## Как выкатывается
1. Push в `main` запускает `.github/workflows/deploy.yml`.
2. Тесты бэкенда и проверка фронтенда → сборка образа (`Dockerfile`) → публикация в GHCR с тегами `sha-<коммит>` и `latest`.
3. По SSH на сервер копируются `compose.yml`, `Caddyfile`, `backup.sh`, `restore.sh`; в `.env` прописывается новый `APP_TAG`; `docker compose pull && up -d`.
4. Миграции применяет само приложение при старте (`CARGO_MIGRATE_ON_START=true` по умолчанию).
5. Проверка: `GET /api/v1/health` отвечает `{"status":"ok"}` в течение 90 с.

Ветки, кроме `main`, и pull request'ы проходят только проверку (`ci.yml`), без выкатки.

## Секреты репозитория
| Секрет | Что это |
|---|---|
| `VPS_HOST` | IP или имя сервера |
| `VPS_USER` | пользователь SSH (`root`) |
| `VPS_SSH_KEY` | приватный ключ деплоя (локально `~/.ssh/cargo_crm_deploy`, публичная часть добавлена в `authorized_keys` сервера) |
| `VPS_KNOWN_HOSTS` | строка `ssh-keyscan -t ed25519 <host>` для проверки отпечатка сервера |

Образ в GHCR приватный; сервер логинится в реестр коротким токеном workflow во время выкатки.

## Первый запуск на новом сервере
```
apt-get update && curl -fsSL https://get.docker.com | sh
fallocate -l 2G /swapfile && chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile && echo '/swapfile none swap sw 0 0' >> /etc/fstab
ufw allow 80/tcp && ufw allow 443/tcp
mkdir -p /opt/cargo-crm/backups
# .env по образцу deploy/.env.example, пароль: openssl rand -hex 24
```
Дальше — обычная выкатка из CI. Первого администратора создают демо-данные или `docker compose exec app /cargo create-admin -login admin -name "Имя" -password '...'`.

## Домен и HTTPS
В `/opt/cargo-crm/.env`: `SITE_ADDRESS=crm.example.kz`, `SECURE_COOKIES=true`; DNS A-запись на IP сервера; `docker compose up -d`. Caddy получит сертификат Let's Encrypt сам.

## Резервные копии
- `backup.sh` снимает `pg_dump` в `backups/cargo-ДАТА.sql.gz`, хранит 14 дней. Крон на сервере: `30 2 * * * /opt/cargo-crm/backup.sh >> /opt/cargo-crm/backups/backup.log 2>&1`.
- `restore.sh <файл>` останавливает приложение, пересоздаёт базу и загружает дамп. Проверка восстановления — сценарий приёмки AT-13.

## Полезное на сервере
```
cd /opt/cargo-crm
docker compose ps
docker compose logs -f app
docker compose exec db psql -U cargo cargo
```
