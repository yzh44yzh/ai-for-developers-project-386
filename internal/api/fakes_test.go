package api_test

import (
	"context"
	"sort"

	"github.com/yzh44yzh/bookmeet/internal/domain"
)

// fakeUserStore is an in-memory domain.UserStore for handler tests.
type fakeUserStore struct {
	byID map[domain.UserID]domain.User
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{byID: make(map[domain.UserID]domain.User)}
}

func (f *fakeUserStore) Add(_ context.Context, u domain.User) error {
	if _, ok := f.byID[u.ID]; ok {
		return domain.ErrUserIDTaken
	}
	for _, existing := range f.byID {
		if existing.Email == u.Email {
			return domain.ErrEmailTaken
		}
	}
	f.byID[u.ID] = u
	return nil
}

func (f *fakeUserStore) Get(_ context.Context, id domain.UserID) (domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (f *fakeUserStore) GetByEmail(_ context.Context, email string) (domain.User, error) {
	for _, u := range f.byID {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

// fakeMeetingStore is an in-memory domain.MeetingStore for handler tests.
type fakeMeetingStore struct {
	meetings map[domain.MeetingID]domain.Meeting
}

func newFakeMeetingStore() *fakeMeetingStore {
	return &fakeMeetingStore{meetings: make(map[domain.MeetingID]domain.Meeting)}
}

func (f *fakeMeetingStore) Create(_ context.Context, m domain.Meeting) error {
	f.meetings[m.ID()] = m
	return nil
}

func (f *fakeMeetingStore) Get(_ context.Context, id domain.MeetingID) (domain.Meeting, error) {
	m, ok := f.meetings[id]
	if !ok {
		return domain.Meeting{}, domain.ErrMeetingNotFound
	}
	return m, nil
}

func (f *fakeMeetingStore) List(_ context.Context) ([]domain.Meeting, error) {
	out := make([]domain.Meeting, 0, len(f.meetings))
	for _, m := range f.meetings {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Start().Equal(out[j].Start()) {
			return out[i].ID() < out[j].ID()
		}
		return out[i].Start().Before(out[j].Start())
	})
	return out, nil
}

func (f *fakeMeetingStore) Update(_ context.Context, id domain.MeetingID, fn func(*domain.Meeting) error) error {
	m, ok := f.meetings[id]
	if !ok {
		return domain.ErrMeetingNotFound
	}
	if err := fn(&m); err != nil {
		return err
	}
	f.meetings[id] = m
	return nil
}
