# AGENTS.md

## Status

**BookMeet** is a web app to book time in calendar for meetings.

## Layout

- `cmd/bookmeet/main.go` — entrypoint, serves on `:8080`, mounts `web.NewMux` (HTML login) next to `api.NewMux` on a root mux. Migrates and pings PostgreSQL at startup (`DATABASE_URL`, default `postgres://test:test@localhost:5432/testdb?sslmode=disable`).
- `cmd/seed/main.go` — dev seeder: inserts Users from a fixed roster (`-users`, default 2, max 10) with random Meetings each (`-meetings`, default 3). Skips users whose email already exists (rerun on a seeded DB is a no-op). Same DB wiring as the main binary.
- `internal/api/` — JSON HTTP API over the domain stores, `NewMux(users, meetings)` registers all routes. Users: `POST /users`, `GET /users/{id}`, `GET /users?email=`. Meetings: `POST /meetings`, `GET /meetings[/{id}]`, `PATCH /meetings/{id}`, `POST /meetings/{id}/cancel`, `POST/DELETE /meetings/{id}/guests[/{user_id}]`. Errors are `{"error":"..."}`; 400 bad input, 404 unknown, 409 wrong state; mutations return 204.
- `internal/domain/` — core domain model (see `CONTEXT.md`): User, Meeting with the Draft ⇄ Scheduled → Cancelled lifecycle, store interfaces. Pure Go, `now time.Time` injected into mutating methods.
- `internal/postgres/` — PostgreSQL implementation of the domain stores (pgx/v5); goose migrations in `migrations/`, embedded and applied at startup.
- `internal/web/` — HTML pages (server-rendered `html/template`, embedded): email login (`GET/POST /login`, success redirects to `/meetings`), session-cookie home page (`GET /` → "Hello {first_name} {last_name}", else redirect to `/login`), `GET /meetings` — upcoming (non-frozen) Meetings owned by the logged-in User with all details, Guests resolved to names, ordered by Start ascending, with an edit link per non-Cancelled Meeting, and meeting creation (`GET/POST /meetings/new`: title, datetime-local start parsed in server-local time, duration from presets, description; PRG to `/meetings`). Meeting editing (`GET/POST /meetings/{id}/edit`: Draft edits all four fields, Scheduled edits title/description only; unknown/not-owned/Cancelled/frozen silently redirect to `/meetings`) and cancellation (`POST /meetings/{id}/edit/cancel`, JS-confirmed button on the edit page; PRG to `/meetings`). All web routes are also registered on the root mux in `cmd/bookmeet/main.go` — unregistered paths fall through to the API mux. In-memory sessions (lost on restart), cookie `bookmeet_session`. See `docs/adr/0003-email-login.md`, `docs/adr/0004-my-meetings-page.md`, `docs/adr/0005-create-meeting-form.md` and `docs/adr/0006-edit-meeting-form.md`.
- `.opencode/skills/` — project skills from github.com/mattpocock/skills (installed manually, editable).

## Commands

- Run: `go run ./cmd/bookmeet`
- Seed: `go run ./cmd/seed` — flags `-users` (default 2, max 10), `-meetings` (default 3).
- Build: `go build ./...`
- Check: `go vet ./...`
- Test: `go test ./...` — integration tests in `internal/postgres/` need `TEST_DATABASE_URL` set (they truncate all tables; never point them at data you care about) and skip otherwise.

## Agent skills

### Issue tracker

Issues are tracked in GitHub Issues on this repo, via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Domain docs

Single-context: one `CONTEXT.md` at the repo root plus `docs/adr/`. See `docs/agents/domain.md`.
