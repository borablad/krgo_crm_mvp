-- +goose Up

-- Номера заявок: ЗО-000001. Номер формирует сервис из последовательности в той же транзакции, что и вставку.
create sequence payment_request_number_seq;

create table payment_requests (
    id               bigint        generated always as identity primary key,
    number           text          not null unique,
    organization_id  bigint        not null references organizations (id),
    counterparty_id  bigint        not null references counterparties (id),
    basis            text          not null,
    amount           numeric(18,2) not null check (amount > 0),
    currency         char(3)       not null references currencies (code),
    pay_date         date          not null,
    cashflow_item_id bigint        not null references cashflow_items (id),
    cost_center_id   bigint        not null references cost_centers (id),
    comment          text,
    status           text          not null default 'draft' check (status in ('draft', 'pending', 'approved', 'rejected', 'paid')),
    created_by       bigint        not null references users (id),
    decided_by       bigint        references users (id),
    decided_at       timestamptz,
    paid_at          timestamptz,
    created_at       timestamptz   not null default now(),
    updated_at       timestamptz   not null default now()
);
-- Последовательность привязана к колонке, чтобы очистка таблицы с restart identity сбрасывала и её.
alter sequence payment_request_number_seq owned by payment_requests.number;

create index payment_requests_status_idx on payment_requests (status);
create index payment_requests_pay_date_idx on payment_requests (pay_date);
create index payment_requests_organization_id_idx on payment_requests (organization_id);
create index payment_requests_counterparty_id_idx on payment_requests (counterparty_id);
create index payment_requests_created_by_idx on payment_requests (created_by);

-- История переходов заявки: только добавление.
create table payment_request_history (
    id         bigint      generated always as identity primary key,
    request_id bigint      not null references payment_requests (id),
    from_status text,
    to_status  text        not null,
    user_id    bigint      references users (id),
    at         timestamptz not null default now(),
    comment    text
);
create index payment_request_history_request_id_idx on payment_request_history (request_id);

-- +goose Down
drop table payment_request_history;
drop table payment_requests;
-- Последовательность привязана к колонке (owned by) и удаляется вместе с таблицей.
drop sequence if exists payment_request_number_seq;
