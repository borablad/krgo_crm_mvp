-- +goose Up

-- Пользователи системы. Вход по логину: у кладовщика может не быть почты.
create table users (
    id            bigint generated always as identity primary key,
    login         text        not null unique,
    full_name     text        not null,
    role          text        not null check (role in ('admin', 'manager', 'employee', 'accountant', 'storekeeper')),
    password_hash text        not null,
    is_active     boolean     not null default true,
    created_at    timestamptz not null default now(),
    updated_at    timestamptz not null default now()
);

-- Сессии. В id хранится sha256 от токена из cookie, сам токен в базу не попадает.
create table sessions (
    id           text        primary key,
    user_id      bigint      not null references users (id) on delete cascade,
    created_at   timestamptz not null default now(),
    expires_at   timestamptz not null,
    last_seen_at timestamptz not null default now(),
    user_agent   text,
    ip           text
);
create index sessions_user_id_idx on sessions (user_id);
create index sessions_expires_at_idx on sessions (expires_at);

-- Журнал действий. Пишется в той же транзакции, что и само действие.
create table audit_log (
    id        bigint      generated always as identity primary key,
    at        timestamptz not null default now(),
    user_id   bigint      references users (id),
    action    text        not null,
    entity    text        not null,
    entity_id bigint,
    details   jsonb       not null default '{}'::jsonb
);
create index audit_log_at_idx on audit_log (at desc);
create index audit_log_entity_idx on audit_log (entity, entity_id);

-- +goose Down
drop table audit_log;
drop table sessions;
drop table users;
