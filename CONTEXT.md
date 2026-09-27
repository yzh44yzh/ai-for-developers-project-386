# BookMeet

BookMeet is a web app for booking time in a calendar for meetings.

## Language

### People

**User**:
A person registered in BookMeet. A User has a first name, last name, description, and a unique email.
_Avoid_: account, member

**Owner**:
The single User whose time a Meeting occupies and who controls it: edit details, manage Guests, cancel. Every Meeting has exactly one Owner; the User who creates a Meeting becomes its Owner.
_Avoid_: host, organizer, creator

**Guest**:
A User participating in a Meeting, effective the moment the Owner adds them. A Meeting may have any number of Guests, but never its own Owner, and never the same Guest twice. A Guest can Leave a Meeting.
_Avoid_: invitee, attendee, participant

### Meetings

**Meeting**:
A gathering of Users at a point in time. A Meeting has a title, a Start, a Duration, a description, and a Status. A Meeting cannot start without at least one Guest. All Meetings are visible to all Users.

**Start**:
The timezone-aware instant at which a Meeting begins. Must lie in the future when the Meeting is created.
_Avoid_: datetime, date

**Duration**:
The length of a Meeting in whole minutes; always strictly positive.

### Lifecycle

**Status**:
The lifecycle state of a Meeting: Draft, Scheduled, or Cancelled. Draft and Scheduled follow the number of Guests; Cancelled is chosen by the Owner and is terminal. Editing a Meeting's details never changes its Status. Once a Meeting's Start + Duration has passed, its record is frozen: no edits, no Guests leaving, no cancellation.

**Draft**:
A Meeting with zero Guests. A Draft cannot start; adding the first Guest makes it Scheduled.
_Avoid_: pending

**Scheduled**:
A Meeting with at least one Guest. If its last Guest Leaves or is removed, it becomes a Draft again.
_Avoid_: confirmed, active

**Cancelled**:
A Meeting called off by its Owner, kept as a record. Cancellation is terminal: a Cancelled Meeting is never edited, revived, or deleted.
_Avoid_: deleted

**Leave**:
A Guest removing themselves from a Meeting.
