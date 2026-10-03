# Create-meeting web form: four fields, datetime-local in server-local time, preset durations, dedicated /meetings/new routes

Logged-in Users create Meetings from the browser at `GET /meetings/new`, posting to `POST /meetings/new`. The form carries four fields: **title** (required — the glossary gives every Meeting a title and the JSON API already rejects a missing one, so the web form matches), **start** via `<input type="datetime-local" step="60">` — the browser's native picker, no JavaScript, whole minutes enforced by the step — parsed in the **server's local timezone**, because the input carries no zone and correct cross-zone handling would need JS or a TZ selector (out of scope for a single-machine dev app), **duration** as a `<select>` of 15/30/45/60/90 minutes (UI convenience only; the domain rule — positive whole minutes — is unchanged and still enforced server-side), and **description**. The web routes deliberately avoid the JSON API's `POST /meetings`: a form POST there would shadow it, and Go's mux prefers the more specific pattern, so `/meetings/new` is never captured by the API's `/meetings/{id}`. On success the server creates the Meeting (a Draft — no Guests yet; guest management stays out of scope) and 303-redirects to `/meetings`; on domain rejection the form re-renders with the error text and a 400. No new domain terms were needed.

## Considered Options

- **No title field** (auto-generate one, or allow empty): rejected — contradicts both the glossary ("a Meeting has a title") and the JSON API's rule for the same operation.
- **Number input for duration**: rejected in favour of presets — picking beats typing for the common cases; the JSON API still accepts arbitrary whole minutes.
- **Separate date + time inputs, or free-text start**: rejected — same timezone problem, worse UX, more parse failures.
- **POST the form to `/meetings` with content negotiation**: rejected — couples HTML and JSON in one handler; a dedicated path keeps both boring.
- **Repopulate the form after a validation error**: rejected for now — error text only, keeping the handler simple; a UX refinement that can land later without changing the design.
