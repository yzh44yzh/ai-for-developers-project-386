# Календарь звонков


[![hexlet-check](https://github.com/yzh44yzh/ai-for-developers-project-386/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/yzh44yzh/ai-for-developers-project-386/actions)

Разработайте совместно с ИИ сервис для бронирования календаря

Учебный проект Хекслета: https://ru.hexlet.io/programs/ai-for-developers
Как это должно работать: https://files.hexlet.app/a/2ipc5m

## Стек

- Golang

## Установка

Требования: Go и PostgreSQL (база данных, пользователь и пароль на ваш выбор).

```bash
git clone https://github.com/yzh44yzh/ai-for-developers-project-386.git
cd ai-for-developers-project-386
```

Переменные окружения:

- `DATABASE_URL` — строка подключения к PostgreSQL, например `postgres://test:test@localhost:5432/testdb?sslmode=disable`. Если не задана, используется именно это значение по умолчанию (только для локальной разработки). Миграции применяются автоматически при запуске.

## Использование

```bash
go run ./cmd/bookmeet    # сервис на :8080
go test ./...            # интеграционные тесты требуют TEST_DATABASE_URL, иначе пропускаются
```

---

<details>
<summary>Автоматические тесты Хекслета</summary>

Тесты запускаются на каждый коммит. За запуск отвечает файл `.github/workflows/hexlet-check.yml` — не удаляйте и не переименовывайте ни его, ни репозиторий.

</details>

## О Хекслете

[Хекслет](https://ru.hexlet.io/) — школа программирования: авторские программы обучения с практикой, поддержкой наставников и реальными проектами, которые остаются в резюме. Этот репозиторий — один из таких проектов.
