# Email login: server-rendered HTML, in-memory sessions, PRG redirect

BookMeet's first HTML page is login: one email field, successful iff the email belongs to a registered User. The page is server-rendered with `html/template` in a new `internal/web` package, mounted next to the (still pure-JSON) `internal/api` mux in `cmd/bookmeet`. The login check reuses the existing `domain.UserStore.GetByEmail`; authentication is deliberately *not* a domain concept — `CONTEXT.md` gains no terms, and "a User exists iff its email is registered" is the only rule. On success the server creates an in-memory session (`map[token]UserID`, `crypto/rand` token), sets an `HttpOnly` cookie `bookmeet_session`, and 303-redirects to `/`, which reads the cookie and renders "Hello {first_name} {last_name}" (missing/unknown cookie → redirect to `/login`; visiting `/login` while logged in → redirect to `/`). An unknown email renders "user not exists" and sets no cookie. Sessions vanish on restart and there is no logout; both are accepted as follow-up-sized gaps.

## Considered Options

- **Static HTML + JS fetch against the existing JSON API**: rejected — adds a JS and static-serving concern to a Go-only codebase for no capability gain.
- **Separate frontend app** (React/Vite/etc.): rejected — far too heavy for one input field.
- **Postgres-backed sessions table**: rejected for now — a migration, store, and expiry cleanup are not justified while losing sessions on restart is harmless (re-login is one email field). The session store is unexported and used only by the web handlers, so it can be swapped later without touching routing.
- **HMAC-signed stateless cookie**: rejected — needs secret-key management, and sessions can't be revoked server-side.
- **Render the success page directly from `POST /login`**: rejected — the cookie would be set but never read, making the saved session decorative, and a refresh would resubmit the form. The Post/Redirect/Get flow makes the session observable.
- **Logout in this change**: rejected — not requested; trivial follow-up.
