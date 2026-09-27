package domain

import (
	"context"
	"errors"
)

var ErrMeetingNotFound = errors.New("meeting not found")

// UserStore persists Users.
type UserStore interface {
	Add(ctx context.Context, u User) error
	Get(ctx context.Context, id UserID) (User, error)
}

// MeetingStore persists Meetings. Update loads the Meeting under a lock,
// hands it to fn for mutation, and persists the result atomically.
type MeetingStore interface {
	Create(ctx context.Context, m Meeting) error
	Get(ctx context.Context, id MeetingID) (Meeting, error)
	List(ctx context.Context) ([]Meeting, error)
	Update(ctx context.Context, id MeetingID, fn func(*Meeting) error) error
}
