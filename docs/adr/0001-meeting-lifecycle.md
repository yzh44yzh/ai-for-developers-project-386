# Meeting lifecycle: Draft/Scheduled derived from guests, Cancelled terminal

A Meeting cannot start without Guests, so its Status is the state machine `Draft ⇄ Scheduled → Cancelled`. Draft and Scheduled are *derived* from the number of Guests (zero Guests = Draft, one or more = Scheduled) rather than set by explicit commands; Cancelled is explicit, terminal, and the record is kept forever. There is no Completed state — past-ness is derived from Start + Duration — and no clock-based automation: a Draft whose Start passes is simply dead. Editing a Meeting's details never changes its Status, and once Start + Duration has passed the record is frozen: no edits, no Guests leaving, no cancellation.

## Considered Options

- **Clock-based enforcement** (background job cancels empty Meetings at their Start): rejected — the transition rules enforce the invariant for free; no background jobs needed.
- **Forbid removing the last Guest** (Owner must cancel instead): rejected — deriving the state in both directions is simpler and loses no history.
- **Completed state for past Meetings**: rejected — redundant with the clock.
- **RSVP per Guest** (invited → accepted/declined): rejected for now — immediate membership keeps the model at two entities; can be layered on as a per-Guest attribute later.
