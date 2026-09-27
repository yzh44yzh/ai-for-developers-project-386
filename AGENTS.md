# AGENTS.md

## Status

**BookMeet** is a web app to book time in calendar for meetings.

## Layout

- `cmd/bookmeet/main.go` — entrypoint, serves on `:8080`, wires routes. Migrates and pings PostgreSQL at startup (`DATABASE_URL`, default `postgres://test:test@localhost:5432/testdb?sslmode=disable`).
- `internal/domain/` — core domain model (see `CONTEXT.md`): User, Meeting with the Draft ⇄ Scheduled → Cancelled lifecycle, store interfaces. Pure Go, `now time.Time` injected into mutating methods.
- `internal/postgres/` — PostgreSQL implementation of the domain stores (pgx/v5); goose migrations in `migrations/`, embedded and applied at startup.
- `internal/hello/` — greeting page handler for `GET /hello`; body logger for `POST /hello`.
- `internal/home/` — root page handler for `GET /`.
- `.opencode/skills/` — project skills from github.com/mattpocock/skills (installed manually, editable).

## Commands

- Run: `go run ./cmd/bookmeet`
- Build: `go build ./...`
- Check: `go vet ./...`
- Test: `go test ./...` — integration tests in `internal/postgres/` need `DATABASE_URL` set (they truncate all tables; never point them at data you care about) and skip otherwise.

## Agent skills

### Issue tracker

Issues are tracked in GitHub Issues on this repo, via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Domain docs

Single-context: one `CONTEXT.md` at the repo root plus `docs/adr/`. See `docs/agents/domain.md`.
