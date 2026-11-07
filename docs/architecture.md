# Архитектура Cargo CRM

Модульный монолит: одно Go-приложение, одна база PostgreSQL, фронтенд-SPA встроен в бинарник. Модули разделены пакетами с явными границами, чтобы при росте любой из них (в первую очередь авторизацию) можно было вынести в отдельный сервис без переписывания.

## Стек
| Слой | Выбор | Зачем |
|---|---|---|
| Язык бэкенда | Go 1.27 | один статический бинарник, простой деплой на слабую VPS |
| HTTP и OpenAPI | `net/http` + [Huma v2](https://huma.rocks) (`humago`) | роутинг стандартный, OpenAPI и валидация входа генерируются из Go-структур; из схемы делаем типы для фронта |
| БД | PostgreSQL 17 | транзакции, `CHECK`-ограничения, блокировки строк для складских остатков |
| Доступ к БД | `pgx/v5` + `sqlc` | обычный SQL в файлах, типобезопасный сгенерированный код, без ORM |
| Миграции | `goose` (SQL-файлы, встроены в бинарник) | `cargo migrate` на старте контейнера |
| Деньги и количества | `numeric(18,2)` / `numeric(14,3)` в БД, `shopspring/decimal` в Go, строка в JSON | никаких float |
| Пароли | argon2id (`golang.org/x/crypto`) | |
| Сессии | cookie `cargo_session` (HttpOnly, SameSite=Lax), хранение в таблице `sessions` | SPA на том же домене; проще и безопаснее JWT в localStorage |
| Фронтенд | React 19, TypeScript, Vite, TanStack Router + Query, react-hook-form + zod, shadcn/ui на Tailwind v4 | |
| Типы API на фронте | `openapi-typescript` + `openapi-fetch` из `/openapi.json` | единый источник правды — Go-структуры |
| Тексты интерфейса | `i18next` + `react-i18next`, словарь `src/i18n/ru.ts` | задел под казахский |
| Контейнеры | один образ `app` (node build → go build → distroless), `postgres:17-alpine`, `caddy:2` | три контейнера на VPS с 1 ГБ памяти |
| CI/CD | GitHub Actions: проверка на PR, сборка образа в GHCR и деплой по SSH на push в `main` | VPS ничего не собирает |

## Структура репозитория
```
backend/
  cmd/cargo/            точка входа: cargo serve | migrate | seed | create-admin
  internal/
    app/                сборка приложения: конфиг → БД → модули → роутер
    config/             переменные окружения
    platform/pg/        пул pgx, транзакции, регистрация типов (decimal)
    platform/httpx/     middleware: request id, лог, recover, CSRF-заголовок
    auth/               сессии, пароли, права, middleware «кто вошёл»
    users/              пользователи и роли
    audit/              журнал действий
    directories/        организации, контрагенты, номенклатура, склады и ячейки, статьи ДДС, ЦФО, единицы измерения, валюты
    crm/                клиенты, сделки, этапы воронки, задачи
    treasury/           заявки на оплату, согласование, платёжный календарь, отчёт ДДС
    wms/                поступления, перемещения, остатки, движения
    web/                встроенный фронтенд (go:embed dist)
  db/migrations/        goose: 0001_core.sql, 0002_directories.sql, ...
  db/queries/<module>/  sqlc-запросы
  sqlc.yaml
frontend/
  src/app/              роутер, провайдеры, каркас (меню, шапка)
  src/api/              клиент openapi-fetch, сгенерированные типы schema.d.ts
  src/components/ui/    shadcn/ui
  src/components/       общие: PageHeader, DataTable, EmptyState, StatusChip, CellTag, MoneyText, FormField
  src/features/<module>/ экраны и формы модуля
  src/i18n/ru.ts        все строки интерфейса
  src/lib/              формат дат и денег, утилиты
deploy/
  compose.yml, Caddyfile, backup.sh, .env.example   то, что лежит на VPS в /opt/cargo-crm
  compose.dev.yml       Postgres для локальной разработки
.github/workflows/      ci.yml (проверка), deploy.yml (образ + выкатка)
docs/                   эта документация
```

## Устройство модуля
Каждый модуль — пакет в `internal/<module>` с одинаковым набором файлов:
- `store/` — код sqlc (генерируется из `db/queries/<module>`), только SQL-операции;
- `service.go` — бизнес-правила: проверки, статусы, транзакции, запись в журнал;
- `handlers.go` — операции Huma: вход/выход структур, права, вызов сервиса. Без логики;
- `module.go` — `Register(api huma.API, deps Deps)`.

Модули не ходят в таблицы друг друга напрямую. Если `treasury` нужен контрагент, он вызывает `directories.Service`. Общие вещи (контекст пользователя, транзакция) передаются через `context.Context` и явные аргументы.

## API
- Префикс `/api/v1`. OpenAPI: `/openapi.json`, документация: `/docs`.
- Списки: `GET /api/v1/crm/clients?q=&status=&limit=&offset=` → `{ items: [...], total: N }`.
- Ошибки — RFC 7807 (`application/problem+json`), поле `detail` на русском, ошибки полей в `errors[]` с `location: "body.bin"`. Фронтенд показывает их у поля.
- Мутации требуют заголовок `X-Requested-With: cargo` (защита от CSRF при cookie-сессии) — клиент ставит его всегда.
- Деньги и количества в JSON — строки: `"125000.00"`.
- Имена структур ответов и тел запросов уникальны во всём приложении: Huma строит схему по имени Go-типа и падает на дубликате. Тела запросов и ответов — именованные типы с префиксом модуля: `UsersCreateBody`, `CrmClient`, `TreasuryListResponse`. Анонимные `Body struct{...}` не использовать: из них получаются схемы вида `ListOutputBody1`.
- Время в JSON — RFC 3339 в UTC; интерфейс показывает в Asia/Almaty. Календарные даты (срок задачи, дата оплаты, дата документа) — строка `ГГГГ-ММ-ДД` без `format:"date"` в телах запросов: формат проверяет сервис и отвечает по-русски у поля.
- Необязательная ссылка в ответе (клиент у задачи, дата закрытия) — `0` или пустая строка, не `null`.
- Параметры списков — встроенная экспортируемая структура (`PageQuery`, `ListQuery`): неэкспортируемую Huma молча пропускает, и `limit`/`offset` не доходят до сервиса.
- Тег `default` только у строк и необязательных полей: у `int`/`bool` Huma подставляет его и при явном `0`/`false`. Для числа со значением по умолчанию — указатель и подстановка в хендлере.
- Булевы фильтры в query — строкой `"true"`/`"false"` (`enum`), отсутствие параметра — без фильтра.

## Права
Роли фиксированы: `admin`, `manager` (руководитель), `employee` (сотрудник), `accountant` (бухгалтер), `storekeeper` (кладовщик). Матрица прав — в `auth/permissions.go`, проверка в хендлерах через `auth.Require(ctx, auth.TreasuryApprove)`. Настройки прав через интерфейс в MVP нет, только назначение роли пользователю.

| Право | admin | manager | employee | accountant | storekeeper |
|---|---|---|---|---|---|
| directories.read | ✓ | ✓ | ✓ | ✓ | ✓ |
| directories.write | ✓ | ✓ | | ✓ (ДДС, ЦФО, контрагенты) | ✓ (склады, ячейки, номенклатура) |
| crm.read / crm.write | ✓ | ✓ | ✓ | | |
| treasury.read | ✓ | ✓ | ✓ (свои) | ✓ | |
| treasury.create | ✓ | ✓ | ✓ | ✓ | |
| treasury.approve | ✓ | ✓ | | | |
| wms.read | ✓ | ✓ | | ✓ | ✓ |
| wms.write (документы, проведение) | ✓ | | | | ✓ |
| users.manage, audit.read | ✓ | | | | |

## База данных
Соглашения: `snake_case`, таблицы во множественном числе, `id bigint generated always as identity`, `created_at`/`updated_at timestamptz`, внешние ключи с индексами, справочники с флагом `is_active` вместо удаления. Номера документов — отдельные последовательности, формат `ЗО-000001` (заявка), `ПС-000001` (поступление), `ПМ-000001` (перемещение).

Таблицы по модулям (поддерживать актуальным):
- **core:** `users`, `sessions`, `audit_log`
- **directories:** `organizations`, `counterparties`, `items`, `units`, `warehouses`, `cells`, `cashflow_items`, `cost_centers`, `currencies`
- **crm:** `clients` (БИН необязателен), `deals` (сумма в тенге, этап `new…won/lost`), `deal_stage_history` (только добавление), `tasks` (привязка к клиенту/сделке необязательна)
- **treasury:** `payment_requests`, `payment_request_history`
- **wms:** `receipts` (черновик → проведено, номер `ПС-…`), `receipt_lines` (уникальный товар в документе), `transfers` (проводятся сразу, номер `ПМ-…`), `stock_movements` (журнал, только добавление; ссылка на документ по `document_type`+`document_id`), `stock_balances` (остаток по паре товар+ячейка, `check (qty >= 0)`)

## Инварианты, которые нельзя нарушать
1. **Остаток не бывает отрицательным.** `stock_balances.qty numeric(14,3) check (qty >= 0)`. Любое движение: в одной транзакции `select ... for update` строк остатков → проверка → запись в `stock_movements` (только добавление) → обновление `stock_balances`. Ошибка БД по `CHECK` — последняя линия обороны, а не основная проверка.
2. **Проведение идемпотентно.** Документ блокируется `for update`, проверяется `status = 'draft'`, иначе ошибка «уже проведён». Движения ссылаются на документ (`document_type`, `document_id`).
3. **Статусы заявки — конечный автомат:** `draft → pending → approved | rejected`, `approved → paid` (ручная отметка). Переходы только через `treasury.Service`, каждый пишется в `payment_request_history` с автором и комментарием.
4. **Журнал пишется в той же транзакции**, что и действие (`audit.Record(ctx, tx, ...)`).
5. **БИН/ИИН** — 12 цифр с проверкой контрольной суммы по алгоритму РК; уникальность контрагентов по БИН/ИИН.
6. **Суммы заявок > 0**, валюта — из справочника; календарь и отчёт ДДС считают суммы по каждой валюте отдельно, без пересчёта по курсу.

## Локальная разработка
```
make dev-db        # Postgres 17 в Docker на 5432
make migrate       # применить миграции
make seed          # демо-данные
make run           # бэкенд на :8080 (отдаёт API; фронт в dev берётся с Vite)
cd frontend && npm run dev   # Vite на :5173, прокси /api → :8080
make check         # vet + тесты + typecheck + сборка фронта
make sqlc          # перегенерировать store после правки SQL
make api-types     # перегенерировать типы API для фронта
```

## Что осознанно не делаем в MVP
Микросервисы, очереди, кеш, Kubernetes, тёмная тема, настройка прав через интерфейс, мультиарендность, интеграции (1С, банки, ЭСФ/СНТ), отгрузка и списание со склада, FIFO/партии, offline-ТСД.

## Журнал решений
| Дата | Решение | Причина |
|---|---|---|
| 2026-11-07 | Фронтенд встраивается в Go-бинарник, один контейнер приложения | VPS с 1 ГБ памяти; меньше контейнеров и точек отказа |
| 2026-11-07 | Cookie-сессии вместо JWT | SPA на том же домене; выход «убивает» сессию сразу; нет токенов в localStorage |
| 2026-11-07 | Журнал действий закладываем в ядро сразу, хотя в бэклоге он P1 | стоит дёшево, если есть с первого дня; задним числом дорого |
| 2026-11-07 | Стенд по IP без HTTPS, Caddy стоит заранее | домена пока нет; при появлении HTTPS включается сменой адреса сайта в Caddyfile |
