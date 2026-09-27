# Persistence: PostgreSQL via pgx, relational mapping, goose migrations

The domain model is persisted in PostgreSQL, behind store interfaces defined in `internal/domain` and implemented in `internal/postgres`; the in-memory store is dropped. The mapping is relational — `users`, `meetings`, `meeting_guests` (primary key on the pair) — and Status is *not* a column: per ADR-0001 it is derived in Go, so only facts are stored (`cancelled_at timestamptz`, guest rows). IDs are application-generated UUID strings, keeping the domain types untouched; they are stored as native `uuid` columns, with `::uuid`/`::text` casts at the SQL boundary so the domain never depends on the driver's types. Schema lives in goose migrations, embedded in the binary and applied at startup. Meeting mutations serialize with `SELECT ... FOR UPDATE` inside a store-level `Update(id, fn)` transaction, so the ADR-0001 invariants hold under concurrency; unique constraints backstop email and guest pairs. Access is via pgx/v5 + pgxpool; configuration is a single `DATABASE_URL`.

## Considered Options

- **JSONB snapshot of the Meeting aggregate**: rejected — hides constraints (unique guest pair, owner FK) the relational schema enforces for free.
- **database/sql + lib/pq**: rejected — lib/pq is in maintenance mode.
- **sqlc**: rejected — codegen tooling is more than this project needs.
- **GORM**: rejected — ORM fights the repo's minimal, stdlib-leaning style.
- **DB-generated bigserial IDs**: rejected — integer IDs would ripple through the domain types and tests.
- **Optimistic concurrency (version column)**: rejected — row locking via `FOR UPDATE` achieves the invariant safety with no retry logic leaking to callers.
