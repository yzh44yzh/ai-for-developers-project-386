# AGENTS.md

## Status

**BookMeet** is a web app to book time in calendar for meetings.

## Layout

- `cmd/bookmeet/main.go` — entrypoint, serves on `:8080`, wires routes.
- `internal/hello/` — greeting handler for `GET /hello`.

## Commands

- Run: `go run ./cmd/bookmeet`
- Build: `go build ./...`
- Check: `go vet ./...`
