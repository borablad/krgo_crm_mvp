-- +goose Up

create sequence receipt_number_seq;
create sequence transfer_number_seq;

-- Поступления на склад. Остатки меняет только проведение (status = 'posted').
create table receipts (
    id              bigint generated always as identity primary key,
    number          text        not null unique,
    warehouse_id    bigint      not null references warehouses (id),
    counterparty_id bigint      not null references counterparties (id),
    doc_date        date        not null,
    comment         text,
    status          text        not null default 'draft' check (status in ('draft', 'posted')),
    created_by      bigint      not null references users (id),
    posted_by       bigint      references users (id),
    posted_at       timestamptz,
    created_at      timestamptz not null default now(),
    updated_at      timestamptz not null default now()
);
create index receipts_warehouse_id_idx on receipts (warehouse_id);
create index receipts_counterparty_id_idx on receipts (counterparty_id);
create index receipts_status_idx on receipts (status);
create index receipts_doc_date_idx on receipts (doc_date);

create table receipt_lines (
    id         bigint generated always as identity primary key,
    receipt_id bigint        not null references receipts (id) on delete cascade,
    item_id    bigint        not null references items (id),
    qty        numeric(14,3) not null check (qty > 0),
    unique (receipt_id, item_id)
);
create index receipt_lines_item_id_idx on receipt_lines (item_id);

-- Перемещения между ячейками одного склада (в том числе размещение из ПРИЁМКИ). Проводятся сразу.
create table transfers (
    id           bigint generated always as identity primary key,
    number       text          not null unique,
    warehouse_id bigint        not null references warehouses (id),
    item_id      bigint        not null references items (id),
    qty          numeric(14,3) not null check (qty > 0),
    from_cell_id bigint        not null references cells (id),
    to_cell_id   bigint        not null references cells (id),
    comment      text,
    created_by   bigint        not null references users (id),
    created_at   timestamptz   not null default now(),
    check (from_cell_id <> to_cell_id)
);
create index transfers_warehouse_id_idx on transfers (warehouse_id);
create index transfers_item_id_idx on transfers (item_id);
create index transfers_from_cell_id_idx on transfers (from_cell_id);
create index transfers_to_cell_id_idx on transfers (to_cell_id);
create index transfers_created_at_idx on transfers (created_at);

-- Журнал движений: только добавление.
create table stock_movements (
    id            bigint generated always as identity primary key,
    at            timestamptz   not null default now(),
    warehouse_id  bigint        not null references warehouses (id),
    item_id       bigint        not null references items (id),
    cell_id       bigint        not null references cells (id),
    qty_delta     numeric(14,3) not null check (qty_delta <> 0),
    document_type text          not null check (document_type in ('receipt', 'transfer')),
    document_id   bigint        not null,
    user_id       bigint        references users (id)
);
create index stock_movements_item_at_idx on stock_movements (item_id, at);
create index stock_movements_cell_id_idx on stock_movements (cell_id);
create index stock_movements_document_idx on stock_movements (document_type, document_id);
create index stock_movements_warehouse_id_idx on stock_movements (warehouse_id);

-- Текущие остатки по ячейкам. check (qty >= 0) — последняя линия обороны; основная проверка в сервисе.
create table stock_balances (
    item_id      bigint        not null references items (id),
    warehouse_id bigint        not null references warehouses (id),
    cell_id      bigint        not null references cells (id),
    qty          numeric(14,3) not null check (qty >= 0),
    updated_at   timestamptz   not null default now(),
    primary key (item_id, cell_id)
);
create index stock_balances_warehouse_id_idx on stock_balances (warehouse_id);
create index stock_balances_cell_id_idx on stock_balances (cell_id);

-- +goose Down
drop table stock_balances;
drop table stock_movements;
drop table transfers;
drop table receipt_lines;
drop table receipts;
drop sequence transfer_number_seq;
drop sequence receipt_number_seq;
