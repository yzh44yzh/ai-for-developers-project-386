package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yzh44yzh/bookmeet/internal/domain"
	"github.com/yzh44yzh/bookmeet/internal/postgres"
)

// testStores connects to the database named by TEST_DATABASE_URL, migrates
// it, and truncates all tables. Never point this at a database with data
// you care about.
func testStores(t *testing.T) (domain.UserStore, domain.MeetingStore) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()

	if err := postgres.Migrate(dsn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "TRUNCATE meeting_guests, meetings, users"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return postgres.NewUsers(pool), postgres.NewMeetings(pool)
}

var now = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

func mustUser(t *testing.T, users domain.UserStore, email string) domain.UserID {
	t.Helper()
	id := domain.NewUserID()
	u := domain.User{ID: id, FirstName: "F", LastName: "L", Email: email}
	if err := users.Add(context.Background(), u); err != nil {
		t.Fatalf("Add user: %v", err)
	}
	return id
}

func mustMeeting(t *testing.T, ownerID domain.UserID) domain.Meeting {
	t.Helper()
	m, err := domain.NewMeeting(domain.NewMeetingID(), ownerID, "Sync", now.Add(time.Hour), 30*time.Minute, "weekly", now)
	if err != nil {
		t.Fatalf("NewMeeting: %v", err)
	}
	return m
}

func TestUserAddAndGet(t *testing.T) {
	users, _ := testStores(t)
	ctx := context.Background()

	ada := domain.User{ID: domain.NewUserID(), FirstName: "Ada", LastName: "Lovelace", Description: "First programmer", Email: "ada@example.com"}
	if err := users.Add(ctx, ada); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := users.Get(ctx, ada.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != ada {
		t.Errorf("got %+v, want %+v", got, ada)
	}
}

func TestUserGetNotFound(t *testing.T) {
	users, _ := testStores(t)

	_, err := users.Get(context.Background(), domain.NewUserID())
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}
}

func TestUserGetByEmail(t *testing.T) {
	users, _ := testStores(t)
	ctx := context.Background()

	ada := domain.User{ID: domain.NewUserID(), FirstName: "Ada", LastName: "Lovelace", Description: "First programmer", Email: "ada@example.com"}
	if err := users.Add(ctx, ada); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := users.GetByEmail(ctx, ada.Email)
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got != ada {
		t.Errorf("got %+v, want %+v", got, ada)
	}
}

func TestUserGetByEmailNotFound(t *testing.T) {
	users, _ := testStores(t)

	_, err := users.GetByEmail(context.Background(), "nobody@example.com")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}
}

func TestUserRejectsDuplicates(t *testing.T) {
	users, _ := testStores(t)
	ctx := context.Background()

	id := domain.NewUserID()
	if err := users.Add(ctx, domain.User{ID: id, Email: "ada@example.com"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := users.Add(ctx, domain.User{ID: id, Email: "grace@example.com"}); !errors.Is(err, domain.ErrUserIDTaken) {
		t.Errorf("duplicate ID: want ErrUserIDTaken, got %v", err)
	}
	if err := users.Add(ctx, domain.User{ID: domain.NewUserID(), Email: "ada@example.com"}); !errors.Is(err, domain.ErrEmailTaken) {
		t.Errorf("duplicate email: want ErrEmailTaken, got %v", err)
	}
}

func TestMeetingCreateAndGet(t *testing.T) {
	users, meetings := testStores(t)
	ctx := context.Background()
	ownerID := mustUser(t, users, "owner@example.com")

	m := mustMeeting(t, ownerID)
	if err := meetings.Create(ctx, m); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := meetings.Get(ctx, m.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title() != "Sync" || got.OwnerID() != ownerID || got.Description() != "weekly" {
		t.Errorf("got %+v", got.Snapshot())
	}
	if !got.Start().Equal(m.Start()) || got.Duration() != m.Duration() {
		t.Errorf("got start %v duration %v", got.Start(), got.Duration())
	}
	if got.Status() != domain.Draft {
		t.Errorf("Status = %v, want Draft", got.Status())
	}
}

func TestMeetingCreatePersistsGuests(t *testing.T) {
	users, meetings := testStores(t)
	ctx := context.Background()
	ownerID := mustUser(t, users, "owner@example.com")
	guestID := mustUser(t, users, "guest@example.com")

	m := mustMeeting(t, ownerID)
	if err := m.AddGuest(guestID, now); err != nil {
		t.Fatalf("AddGuest: %v", err)
	}
	if err := meetings.Create(ctx, m); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := meetings.Get(ctx, m.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status() != domain.Scheduled {
		t.Errorf("Status = %v, want Scheduled", got.Status())
	}
	if guests := got.Guests(); len(guests) != 1 || guests[0] != guestID {
		t.Errorf("Guests = %v", guests)
	}
}

func TestMeetingGetNotFound(t *testing.T) {
	_, meetings := testStores(t)

	_, err := meetings.Get(context.Background(), domain.NewMeetingID())
	if !errors.Is(err, domain.ErrMeetingNotFound) {
		t.Fatalf("want ErrMeetingNotFound, got %v", err)
	}
}

func TestMeetingList(t *testing.T) {
	users, meetings := testStores(t)
	ctx := context.Background()
	ownerID := mustUser(t, users, "owner@example.com")

	if err := meetings.Create(ctx, mustMeeting(t, ownerID)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := meetings.Create(ctx, mustMeeting(t, ownerID)); err != nil {
		t.Fatalf("Create: %v", err)
	}

	list, err := meetings.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List returned %d meetings, want 2", len(list))
	}
}

func TestMeetingUpdateAddsAndRemovesGuests(t *testing.T) {
	users, meetings := testStores(t)
	ctx := context.Background()
	ownerID := mustUser(t, users, "owner@example.com")
	guestID := mustUser(t, users, "guest@example.com")

	m := mustMeeting(t, ownerID)
	if err := meetings.Create(ctx, m); err != nil {
		t.Fatalf("Create: %v", err)
	}

	err := meetings.Update(ctx, m.ID(), func(m *domain.Meeting) error {
		return m.AddGuest(guestID, now)
	})
	if err != nil {
		t.Fatalf("Update AddGuest: %v", err)
	}

	got, err := meetings.Get(ctx, m.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status() != domain.Scheduled {
		t.Errorf("Status = %v, want Scheduled", got.Status())
	}
	if guests := got.Guests(); len(guests) != 1 || guests[0] != guestID {
		t.Errorf("Guests = %v", guests)
	}

	err = meetings.Update(ctx, m.ID(), func(m *domain.Meeting) error {
		return m.RemoveGuest(guestID, now)
	})
	if err != nil {
		t.Fatalf("Update RemoveGuest: %v", err)
	}
	got, _ = meetings.Get(ctx, m.ID())
	if got.Status() != domain.Draft {
		t.Errorf("Status = %v, want Draft after last Guest left", got.Status())
	}
}

func TestMeetingUpdateCancelPersists(t *testing.T) {
	users, meetings := testStores(t)
	ctx := context.Background()
	ownerID := mustUser(t, users, "owner@example.com")

	m := mustMeeting(t, ownerID)
	if err := meetings.Create(ctx, m); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := meetings.Update(ctx, m.ID(), func(m *domain.Meeting) error { return m.Cancel(now) }); err != nil {
		t.Fatalf("Update Cancel: %v", err)
	}

	got, err := meetings.Get(ctx, m.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status() != domain.Cancelled || got.CancelledAt() == nil {
		t.Errorf("Status = %v, CancelledAt = %v", got.Status(), got.CancelledAt())
	}

	err = meetings.Update(ctx, m.ID(), func(m *domain.Meeting) error {
		return m.AddGuest(domain.NewUserID(), now)
	})
	if !errors.Is(err, domain.ErrMeetingCancelled) {
		t.Fatalf("AddGuest after cancel: want ErrMeetingCancelled, got %v", err)
	}
}

func TestMeetingUpdateNotFound(t *testing.T) {
	_, meetings := testStores(t)

	err := meetings.Update(context.Background(), domain.NewMeetingID(), func(m *domain.Meeting) error { return nil })
	if !errors.Is(err, domain.ErrMeetingNotFound) {
		t.Fatalf("want ErrMeetingNotFound, got %v", err)
	}
}

func TestMeetingGetPastStartReconstitutes(t *testing.T) {
	users, meetings := testStores(t)
	ctx := context.Background()
	ownerID := mustUser(t, users, "owner@example.com")

	m := mustMeeting(t, ownerID)
	if err := meetings.Create(ctx, m); err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Editing may move the Start into the past (frozen rule is the only
	// barrier); loading that Meeting later must not fail validation.
	pastStart := now.Add(-2 * time.Hour)
	err := meetings.Update(ctx, m.ID(), func(m *domain.Meeting) error {
		return m.EditDetails("Late Sync", pastStart, time.Hour, "", now)
	})
	if err != nil {
		t.Fatalf("Update EditDetails: %v", err)
	}

	got, err := meetings.Get(ctx, m.ID())
	if err != nil {
		t.Fatalf("Get past meeting: %v", err)
	}
	if !got.Start().Equal(pastStart) || got.Title() != "Late Sync" {
		t.Errorf("got %+v", got.Snapshot())
	}
}
