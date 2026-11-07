-- +goose Up

-- Организации группы (наши юрлица): плательщики в заявках на оплату.
create table organizations (
    id         bigint generated always as identity primary key,
    name       text        not null,
    bin        text        not null unique,
    address    text,
    is_active  boolean     not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Контрагенты: поставщики, покупатели и прочие.
create table counterparties (
    id           bigint generated always as identity primary key,
    name         text        not null,
    bin          text        not null unique,
    kind         text        not null check (kind in ('supplier', 'customer', 'other')),
    contact_name text,
    phone        text,
    email        text,
    is_active    boolean     not null default true,
    created_at   timestamptz not null default now(),
    updated_at   timestamptz not null default now()
);

-- Единицы измерения.
create table units (
    id   bigint generated always as identity primary key,
    code text not null unique,
    name text not null
);
insert into units (code, name) values
    ('шт', 'Штука'),
    ('кг', 'Килограмм'),
    ('л', 'Литр'),
    ('м', 'Метр'),
    ('упак', 'Упаковка');

-- Номенклатура. Штрихкод необязателен, но если задан — уникален.
create table items (
    id         bigint generated always as identity primary key,
    name       text        not null,
    sku        text        not null unique,
    unit_id    bigint      not null references units (id),
    barcode    text,
    is_active  boolean     not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);
create unique index items_barcode_uniq on items (barcode) where barcode is not null;

create table warehouses (
    id                  bigint generated always as identity primary key,
    name                text        not null,
    code                text        not null unique,
    responsible_user_id bigint      references users (id),
    is_active           boolean     not null default true,
    created_at          timestamptz not null default now(),
    updated_at          timestamptz not null default now()
);

-- Ячейки склада. is_receiving — служебная ячейка «ПРИЁМКА», создаётся вместе со складом.
create table cells (
    id           bigint generated always as identity primary key,
    warehouse_id bigint      not null references warehouses (id),
    address      text        not null,
    is_receiving boolean     not null default false,
    is_active    boolean     not null default true,
    created_at   timestamptz not null default now(),
    unique (warehouse_id, address)
);
create index cells_warehouse_id_idx on cells (warehouse_id);

-- Статьи движения денежных средств (ДДС).
create table cashflow_items (
    id         bigint generated always as identity primary key,
    name       text        not null,
    kind       text        not null check (kind in ('income', 'expense')),
    is_active  boolean     not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Центры финансовой ответственности (подразделения).
create table cost_centers (
    id         bigint generated always as identity primary key,
    name       text        not null,
    code       text        unique,
    is_active  boolean     not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table currencies (
    code      char(3)  primary key,
    name      text     not null,
    is_active boolean  not null default true
);
insert into currencies (code, name) values
    ('KZT', 'Тенге'),
    ('USD', 'Доллар США'),
    ('EUR', 'Евро'),
    ('RUB', 'Российский рубль');

-- +goose Down
drop table currencies;
drop table cost_centers;
drop table cashflow_items;
drop table cells;
drop table warehouses;
drop table items;
drop table units;
drop table counterparties;
drop table organizations;
