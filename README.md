# Cargo CRM

Внутренняя система для небольшой компании: клиенты и сделки (CRM), заявки на оплату и их согласование (казначейство), приёмка и адресное хранение товара (склад). MVP.

## Запуск для разработки
Нужны Go 1.27+, Node 22+, Docker.

```
make dev-db                  # Postgres в Docker
make migrate && make seed    # схема и демо-данные
make run                     # бэкенд: http://localhost:8080
cd frontend && npm install && npm run dev   # интерфейс: http://localhost:5173
```

Вход с демо-данными: см. вывод `make seed`.

## Документация
- [Архитектура и соглашения](docs/architecture.md)
- [Правила интерфейса](docs/ux.md)
- [Бэклог и статусы задач](docs/backlog.md)
- [Сценарии приёмки](docs/acceptance.md)
- [Деплой](docs/deploy.md)
- [Лог разработки](docs/devlog.md)
