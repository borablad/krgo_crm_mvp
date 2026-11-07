-- +goose Up

-- Клиенты (компании). БИН необязателен, но если задан — 12 цифр; контрольную сумму проверяет сервис.
create table clients (
    id                bigint generated always as identity primary key,
    name              text        not null,
    bin               text        check (bin ~ '^[0-9]{12}$'),
    industry          text,
    region            text,
    contact_name      text,
    contact_position  text,
    phone             text,
    email             text,
    owner_user_id     bigint      not null references users (id),
    status            text        not null default 'potential' check (status in ('potential', 'active', 'inactive')),
    created_at        timestamptz not null default now(),
    updated_at        timestamptz not null default now()
);
create index clients_owner_user_id_idx on clients (owner_user_id);
create index clients_name_idx on clients (lower(name));

-- Сделки. Валюта одна — тенге, поля валюты нет.
create table deals (
    id                  bigint        generated always as identity primary key,
    client_id           bigint        not null references clients (id),
    title               text          not null,
    amount              numeric(18,2) not null check (amount >= 0),
    probability         int           not null default 50 check (probability between 0 and 100),
    expected_close_date date,
    owner_user_id       bigint        not null references users (id),
    stage               text          not null default 'new' check (stage in ('new', 'contact', 'negotiation', 'proposal', 'approval', 'won', 'lost')),
    stage_changed_at    timestamptz   not null default now(),
    created_at          timestamptz   not null default now(),
    updated_at          timestamptz   not null default now()
);
create index deals_client_id_idx on deals (client_id);
create index deals_owner_user_id_idx on deals (owner_user_id);
create index deals_stage_idx on deals (stage);

-- История этапов сделки: только добавление.
create table deal_stage_history (
    id         bigint      generated always as identity primary key,
    deal_id    bigint      not null references deals (id),
    from_stage text,
    to_stage   text        not null,
    changed_by bigint      references users (id),
    changed_at timestamptz not null default now(),
    comment    text
);
create index deal_stage_history_deal_id_idx on deal_stage_history (deal_id);

-- Задачи сотрудников.
create table tasks (
    id               bigint generated always as identity primary key,
    title            text        not null,
    description      text,
    assignee_user_id bigint      not null references users (id),
    due_date         date        not null,
    status           text        not null default 'open' check (status in ('open', 'done', 'cancelled')),
    client_id        bigint      references clients (id),
    deal_id          bigint      references deals (id),
    created_by       bigint      references users (id),
    done_at          timestamptz,
    created_at       timestamptz not null default now(),
    updated_at       timestamptz not null default now()
);
create index tasks_assignee_user_id_idx on tasks (assignee_user_id);
create index tasks_status_idx on tasks (status);
create index tasks_due_date_idx on tasks (due_date);
create index tasks_client_id_idx on tasks (client_id);
create index tasks_deal_id_idx on tasks (deal_id);

-- +goose Down
drop table tasks;
drop table deal_stage_history;
drop table deals;
drop table clients;
