# AGENTS.md

## Status

**BookMeet** is a web app to book time in calendar for meetings.

## Layout

- `cmd/bookmeet/main.go` — entrypoint, serves on `:8080`, wires routes.
- `internal/hello/` — greeting page handler for `GET /hello`; body logger for `POST /hello`.
- `internal/home/` — root page handler for `GET /`.
- `.opencode/skills/` — project skills from github.com/mattpocock/skills (installed manually, editable).

## Commands

- Run: `go run ./cmd/bookmeet`
- Build: `go build ./...`
- Check: `go vet ./...`
- Test: `go test ./...`

## Agent skills

### Issue tracker

Issues are tracked in GitHub Issues on this repo, via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Domain docs

Single-context: one `CONTEXT.md` at the repo root plus `docs/adr/`. See `docs/agents/domain.md`.
