package domain

import (
	"errors"
	"slices"
	"time"
)

var (
	ErrStartInPast      = errors.New("start must be in the future")
	ErrInvalidDuration  = errors.New("duration must be a positive whole number of minutes")
	ErrOwnerAsGuest     = errors.New("owner cannot be a guest of their own meeting")
	ErrDuplicateGuest   = errors.New("user is already a guest of this meeting")
	ErrNotGuest         = errors.New("user is not a guest of this meeting")
	ErrMeetingCancelled = errors.New("meeting is cancelled")
	ErrMeetingFrozen    = errors.New("meeting is in the past and its record is frozen")
	ErrInvalidSnapshot  = errors.New("invalid meeting snapshot")
)

type MeetingID string

// Status is the lifecycle state of a Meeting.
type Status string

const (
	Draft     Status = "draft"
	Scheduled Status = "scheduled"
	Cancelled Status = "cancelled"
)

// Meeting is a gathering of Users at a point in time.
type Meeting struct {
	id          MeetingID
	ownerID     UserID
	title       string
	start       time.Time
	duration    time.Duration
	description string
	guests      map[UserID]struct{}
	cancelledAt *time.Time
}

// NewMeeting creates a Meeting owned by ownerID. The Start must lie in
// the future and the Duration must be a positive whole number of minutes.
func NewMeeting(id MeetingID, ownerID UserID, title string, start time.Time, duration time.Duration, description string, now time.Time) (Meeting, error) {
	if !start.After(now) {
		return Meeting{}, ErrStartInPast
	}
	if duration <= 0 || duration%time.Minute != 0 {
		return Meeting{}, ErrInvalidDuration
	}
	return Meeting{
		id:          id,
		ownerID:     ownerID,
		title:       title,
		start:       start,
		duration:    duration,
		description: description,
		guests:      make(map[UserID]struct{}),
	}, nil
}

func (m Meeting) ID() MeetingID           { return m.id }
func (m Meeting) OwnerID() UserID         { return m.ownerID }
func (m Meeting) Title() string           { return m.title }
func (m Meeting) Start() time.Time        { return m.start }
func (m Meeting) Duration() time.Duration { return m.duration }
func (m Meeting) Description() string     { return m.description }

// CancelledAt reports when the Meeting was cancelled, or nil if it was not.
func (m Meeting) CancelledAt() *time.Time { return m.cancelledAt }

// Status is derived: Cancelled if the Owner cancelled, Draft while the
// Meeting has no Guests, Scheduled otherwise.
func (m Meeting) Status() Status {
	if m.cancelledAt != nil {
		return Cancelled
	}
	if len(m.guests) == 0 {
		return Draft
	}
	return Scheduled
}

// Guests returns the Meeting's Guests in stable order.
func (m Meeting) Guests() []UserID {
	guests := make([]UserID, 0, len(m.guests))
	for id := range m.guests {
		guests = append(guests, id)
	}
	slices.Sort(guests)
	return guests
}

// frozen reports whether the Meeting's Start + Duration has passed; a
// frozen record rejects all mutations.
func (m Meeting) frozen(now time.Time) bool {
	return !now.Before(m.start.Add(m.duration))
}

// checkMutable enforces the two barriers every mutation shares: a frozen
// (past) record and a Cancelled Meeting both reject all changes.
func (m Meeting) checkMutable(now time.Time) error {
	if m.frozen(now) {
		return ErrMeetingFrozen
	}
	if m.cancelledAt != nil {
		return ErrMeetingCancelled
	}
	return nil
}

// AddGuest adds a User to the Meeting, effective immediately. The Owner can
// never be a Guest, and the same User cannot be a Guest twice.
func (m *Meeting) AddGuest(id UserID, now time.Time) error {
	if err := m.checkMutable(now); err != nil {
		return err
	}
	if id == m.ownerID {
		return ErrOwnerAsGuest
	}
	if _, ok := m.guests[id]; ok {
		return ErrDuplicateGuest
	}
	m.guests[id] = struct{}{}
	return nil
}

// RemoveGuest removes a Guest from the Meeting: either the Owner managing
// Guests or the Guest leaving. If the last Guest leaves, the Meeting
// reverts to Draft.
func (m *Meeting) RemoveGuest(id UserID, now time.Time) error {
	if err := m.checkMutable(now); err != nil {
		return err
	}
	if _, ok := m.guests[id]; !ok {
		return ErrNotGuest
	}
	delete(m.guests, id)
	return nil
}

// Cancel calls the Meeting off. Cancellation is terminal: a Cancelled
// Meeting is never edited, revived, or deleted.
func (m *Meeting) Cancel(now time.Time) error {
	if err := m.checkMutable(now); err != nil {
		return err
	}
	m.cancelledAt = &now
	return nil
}

// EditDetails replaces the Meeting's title, Start, Duration and description.
// Editing never changes the Status. Unlike at creation, the Start may lie
// in the past — the frozen rule is the only barrier to edits.
func (m *Meeting) EditDetails(title string, start time.Time, duration time.Duration, description string, now time.Time) error {
	if err := m.checkMutable(now); err != nil {
		return err
	}
	if duration <= 0 || duration%time.Minute != 0 {
		return ErrInvalidDuration
	}
	m.title = title
	m.start = start
	m.duration = duration
	m.description = description
	return nil
}

// MeetingSnapshot is the persistable state of a Meeting. It exists for
// store adapters; application code should use NewMeeting and the Meeting
// methods instead.
type MeetingSnapshot struct {
	ID          MeetingID
	OwnerID     UserID
	Title       string
	Start       time.Time
	Duration    time.Duration
	Description string
	Guests      []UserID
	CancelledAt *time.Time
}

// Snapshot captures the Meeting's persistable state.
func (m Meeting) Snapshot() MeetingSnapshot {
	return MeetingSnapshot{
		ID:          m.id,
		OwnerID:     m.ownerID,
		Title:       m.title,
		Start:       m.start,
		Duration:    m.duration,
		Description: m.description,
		Guests:      m.Guests(),
		CancelledAt: m.cancelledAt,
	}
}

// MeetingFromSnapshot reconstitutes a Meeting from persisted state. It
// exists for store adapters. Creation-time rules are not re-checked (a
// persisted Meeting may have a past Start), but structural invariants are:
// a snapshot that violates them is corrupt and returns ErrInvalidSnapshot.
func MeetingFromSnapshot(s MeetingSnapshot) (Meeting, error) {
	if s.ID == "" || s.OwnerID == "" {
		return Meeting{}, ErrInvalidSnapshot
	}
	if s.Duration <= 0 || s.Duration%time.Minute != 0 {
		return Meeting{}, ErrInvalidSnapshot
	}
	guests := make(map[UserID]struct{}, len(s.Guests))
	for _, id := range s.Guests {
		if id == s.OwnerID {
			return Meeting{}, ErrInvalidSnapshot
		}
		guests[id] = struct{}{}
	}
	return Meeting{
		id:          s.ID,
		ownerID:     s.OwnerID,
		title:       s.Title,
		start:       s.Start,
		duration:    s.Duration,
		description: s.Description,
		guests:      guests,
		cancelledAt: s.CancelledAt,
	}, nil
}
