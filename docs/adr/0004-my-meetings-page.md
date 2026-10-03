# My-meetings page: owner-scoped store query, past Meetings hidden, Guests resolved in the handler

The first authenticated view is `GET /meetings`: the upcoming Meetings owned by the logged-in User with all details (title, Start, Duration, description, derived Status, cancelled-at when applicable) and Guests rendered as "First Last \<email>"; login now redirects there, while `/` keeps the greeting and links to it. Ownership filtering lives in the store — `domain.MeetingStore` gained `ListByOwner(ctx, owner)` (SQL `WHERE owner_id = ...`, ordered `start, id`) — rather than filtering `List()` in the handler, because answering "Meetings owned by X" is the store's job and more owner-scoped callers are expected. Past Meetings are hidden in the web layer via the newly exported `Meeting.Frozen(now)`: past-ness stays a domain rule (ADR-0001), merely applied as a view concern; no new glossary terms were needed. Guests are resolved one by one via `UserStore.Get`; an unresolvable Guest ID renders raw.

## Considered Options

- **Filter `List()` in the web handler**: rejected — loads every Meeting forever and makes each future caller re-implement the ownership filter; a `WHERE` clause is cheaper than the interface churn.
- **Meetings replace the greeting at `/`**: rejected — a dedicated URL keeps the greeting page intact and makes "redirect after login" explicit.
- **Hide Cancelled Meetings as well**: rejected — only past (frozen) ones are hidden; an upcoming Cancelled Meeting is still useful information, and the Status line says so.
- **Show raw Guest UUIDs or just a count**: rejected — "all details" should be human-readable; N+1 lookups are trivial at this scale, and if they ever matter, batch resolution belongs behind the store, not in the template.
- **Keep `frozen` unexported and re-derive past-ness in the handler**: rejected — duplicating "Start + Duration has passed" in the web layer would fork a domain rule; exporting the method costs nothing (`checkMutable` uses the same one).
