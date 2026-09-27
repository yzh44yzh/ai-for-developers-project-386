package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/yzh44yzh/bookmeet/internal/domain"
)

var now = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

func newMeeting(t *testing.T) domain.Meeting {
	t.Helper()
	m, err := domain.NewMeeting("m1", "u-owner", "Sync", now.Add(time.Hour), 30*time.Minute, "weekly", now)
	if err != nil {
		t.Fatalf("NewMeeting: %v", err)
	}
	return m
}

func TestNewMeetingMakesCreatingUserTheOwner(t *testing.T) {
	m := newMeeting(t)

	if m.OwnerID() != "u-owner" {
		t.Errorf("OwnerID = %v", m.OwnerID())
	}
	if m.Status() != domain.Draft {
		t.Errorf("Status = %v, want Draft (no Guests yet)", m.Status())
	}
	if m.Title() != "Sync" || m.Description() != "weekly" {
		t.Errorf("got title %q description %q", m.Title(), m.Description())
	}
	if m.Duration() != 30*time.Minute {
		t.Errorf("Duration = %v", m.Duration())
	}
}

func TestNewMeetingRejectsStartInPast(t *testing.T) {
	_, err := domain.NewMeeting("m1", "u-owner", "Sync", now.Add(-time.Hour), 30*time.Minute, "", now)
	if !errors.Is(err, domain.ErrStartInPast) {
		t.Fatalf("want ErrStartInPast, got %v", err)
	}
}

func TestNewMeetingRejectsBadDuration(t *testing.T) {
	for _, d := range []time.Duration{0, -time.Minute, 90 * time.Second} {
		_, err := domain.NewMeeting("m1", "u-owner", "Sync", now.Add(time.Hour), d, "", now)
		if !errors.Is(err, domain.ErrInvalidDuration) {
			t.Errorf("duration %v: want ErrInvalidDuration, got %v", d, err)
		}
	}
}

func TestAddGuestSchedulesMeeting(t *testing.T) {
	m := newMeeting(t)

	if err := m.AddGuest("u-guest", now); err != nil {
		t.Fatalf("AddGuest: %v", err)
	}
	if m.Status() != domain.Scheduled {
		t.Errorf("Status = %v, want Scheduled", m.Status())
	}
	guests := m.Guests()
	if len(guests) != 1 || guests[0] != "u-guest" {
		t.Errorf("Guests = %v", guests)
	}
}

func TestAddGuestOwnerCannotBeGuest(t *testing.T) {
	m := newMeeting(t)

	err := m.AddGuest("u-owner", now)
	if !errors.Is(err, domain.ErrOwnerAsGuest) {
		t.Fatalf("want ErrOwnerAsGuest, got %v", err)
	}
}

func TestAddGuestRejectsDuplicates(t *testing.T) {
	m := newMeeting(t)
	if err := m.AddGuest("u-guest", now); err != nil {
		t.Fatalf("AddGuest: %v", err)
	}

	err := m.AddGuest("u-guest", now)
	if !errors.Is(err, domain.ErrDuplicateGuest) {
		t.Fatalf("want ErrDuplicateGuest, got %v", err)
	}
}

func TestRemoveGuest(t *testing.T) {
	m := newMeeting(t)
	if err := m.AddGuest("u-g1", now); err != nil {
		t.Fatalf("AddGuest: %v", err)
	}
	if err := m.AddGuest("u-g2", now); err != nil {
		t.Fatalf("AddGuest: %v", err)
	}

	if err := m.RemoveGuest("u-g1", now); err != nil {
		t.Fatalf("RemoveGuest: %v", err)
	}
	if got := m.Guests(); len(got) != 1 || got[0] != "u-g2" {
		t.Errorf("Guests = %v", got)
	}
	if m.Status() != domain.Scheduled {
		t.Errorf("Status = %v, want Scheduled", m.Status())
	}
}

func TestRemoveLastGuestRevertsToDraft(t *testing.T) {
	m := newMeeting(t)
	if err := m.AddGuest("u-guest", now); err != nil {
		t.Fatalf("AddGuest: %v", err)
	}

	if err := m.RemoveGuest("u-guest", now); err != nil {
		t.Fatalf("RemoveGuest: %v", err)
	}
	if m.Status() != domain.Draft {
		t.Errorf("Status = %v, want Draft after last Guest left", m.Status())
	}
}

func TestRemoveGuestRejectsNonGuest(t *testing.T) {
	m := newMeeting(t)

	err := m.RemoveGuest("u-stranger", now)
	if !errors.Is(err, domain.ErrNotGuest) {
		t.Fatalf("want ErrNotGuest, got %v", err)
	}
}

func TestCancelScheduledMeeting(t *testing.T) {
	m := newMeeting(t)
	if err := m.AddGuest("u-guest", now); err != nil {
		t.Fatalf("AddGuest: %v", err)
	}

	if err := m.Cancel(now); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if m.Status() != domain.Cancelled {
		t.Errorf("Status = %v, want Cancelled (wins over guest count)", m.Status())
	}
	if got := m.Guests(); len(got) != 1 {
		t.Errorf("Guests = %v, want the record kept", got)
	}
}

func TestCancelDraftMeeting(t *testing.T) {
	m := newMeeting(t)

	if err := m.Cancel(now); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if m.Status() != domain.Cancelled {
		t.Errorf("Status = %v, want Cancelled", m.Status())
	}
}

func TestCancelIsTerminal(t *testing.T) {
	m := newMeeting(t)
	if err := m.Cancel(now); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	if err := m.Cancel(now); !errors.Is(err, domain.ErrMeetingCancelled) {
		t.Errorf("second Cancel: want ErrMeetingCancelled, got %v", err)
	}
	if err := m.AddGuest("u-guest", now); !errors.Is(err, domain.ErrMeetingCancelled) {
		t.Errorf("AddGuest after cancel: want ErrMeetingCancelled, got %v", err)
	}
	if err := m.RemoveGuest("u-guest", now); !errors.Is(err, domain.ErrMeetingCancelled) {
		t.Errorf("RemoveGuest after cancel: want ErrMeetingCancelled, got %v", err)
	}
}

func TestEditDetails(t *testing.T) {
	m := newMeeting(t)
	newStart := now.Add(2 * time.Hour)

	err := m.EditDetails("Retro", newStart, 45*time.Minute, "biweekly", now)
	if err != nil {
		t.Fatalf("EditDetails: %v", err)
	}
	if m.Title() != "Retro" || m.Description() != "biweekly" {
		t.Errorf("got title %q description %q", m.Title(), m.Description())
	}
	if !m.Start().Equal(newStart) || m.Duration() != 45*time.Minute {
		t.Errorf("got start %v duration %v", m.Start(), m.Duration())
	}
}

func TestEditDetailsNeverChangesStatus(t *testing.T) {
	m := newMeeting(t)
	if err := m.AddGuest("u-guest", now); err != nil {
		t.Fatalf("AddGuest: %v", err)
	}

	if err := m.EditDetails("Retro", now.Add(2*time.Hour), 45*time.Minute, "", now); err != nil {
		t.Fatalf("EditDetails: %v", err)
	}
	if m.Status() != domain.Scheduled {
		t.Errorf("Status = %v, want Scheduled", m.Status())
	}
}

func TestEditDetailsValidatesDuration(t *testing.T) {
	m := newMeeting(t)

	err := m.EditDetails("Retro", now.Add(time.Hour), 90*time.Second, "", now)
	if !errors.Is(err, domain.ErrInvalidDuration) {
		t.Fatalf("want ErrInvalidDuration, got %v", err)
	}
}

func TestEditDetailsMidMeetingKeepsPastStart(t *testing.T) {
	m := newMeeting(t)
	midMeeting := now.Add(time.Hour + 15*time.Minute)

	// Title-only edit mid-meeting: relative to now the Start is already in
	// the past, and that is fine — the frozen rule is the only edit barrier.
	if err := m.EditDetails("Retro", m.Start(), m.Duration(), "", midMeeting); err != nil {
		t.Fatalf("EditDetails: %v", err)
	}
	if m.Title() != "Retro" {
		t.Errorf("Title = %q", m.Title())
	}
}

func TestEditDetailsRejectsCancelled(t *testing.T) {
	m := newMeeting(t)
	if err := m.Cancel(now); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	err := m.EditDetails("Retro", now.Add(time.Hour), 45*time.Minute, "", now)
	if !errors.Is(err, domain.ErrMeetingCancelled) {
		t.Fatalf("want ErrMeetingCancelled, got %v", err)
	}
}

// afterEnd is past the meeting created by newMeeting (start now+1h, 30min).
var afterEnd = now.Add(2 * time.Hour)

func TestFrozenPastRejectsMutations(t *testing.T) {
	m := newMeeting(t)
	if err := m.AddGuest("u-guest", now); err != nil {
		t.Fatalf("AddGuest: %v", err)
	}

	if err := m.AddGuest("u-g2", afterEnd); !errors.Is(err, domain.ErrMeetingFrozen) {
		t.Errorf("AddGuest: want ErrMeetingFrozen, got %v", err)
	}
	if err := m.RemoveGuest("u-guest", afterEnd); !errors.Is(err, domain.ErrMeetingFrozen) {
		t.Errorf("RemoveGuest: want ErrMeetingFrozen, got %v", err)
	}
	if err := m.Cancel(afterEnd); !errors.Is(err, domain.ErrMeetingFrozen) {
		t.Errorf("Cancel: want ErrMeetingFrozen, got %v", err)
	}
	if err := m.EditDetails("Retro", now.Add(3*time.Hour), time.Hour, "", afterEnd); !errors.Is(err, domain.ErrMeetingFrozen) {
		t.Errorf("EditDetails: want ErrMeetingFrozen, got %v", err)
	}
}

func TestMeetingInProgressIsNotFrozen(t *testing.T) {
	m := newMeeting(t)
	midMeeting := now.Add(time.Hour + 15*time.Minute)

	if err := m.AddGuest("u-guest", midMeeting); err != nil {
		t.Fatalf("AddGuest during the meeting: %v", err)
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	m := newMeeting(t)
	if err := m.AddGuest("u-guest", now); err != nil {
		t.Fatalf("AddGuest: %v", err)
	}
	if err := m.Cancel(now); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	r, err := domain.MeetingFromSnapshot(m.Snapshot())
	if err != nil {
		t.Fatalf("MeetingFromSnapshot: %v", err)
	}
	if r.Status() != m.Status() || r.Status() != domain.Cancelled {
		t.Errorf("Status = %v, want %v (Cancelled)", r.Status(), m.Status())
	}
	if r.ID() != m.ID() || r.OwnerID() != m.OwnerID() || r.Title() != m.Title() {
		t.Errorf("identity fields differ: %+v", r.Snapshot())
	}
	if !r.Start().Equal(m.Start()) || r.Duration() != m.Duration() || r.Description() != m.Description() {
		t.Errorf("detail fields differ: %+v", r.Snapshot())
	}
	if got := r.Guests(); len(got) != 1 || got[0] != "u-guest" {
		t.Errorf("Guests = %v", got)
	}
	if r.CancelledAt() == nil || !r.CancelledAt().Equal(now) {
		t.Errorf("CancelledAt = %v, want %v", r.CancelledAt(), now)
	}
}

func TestMeetingFromSnapshotRejectsCorrupt(t *testing.T) {
	m := newMeeting(t)

	ownerAsGuest := m.Snapshot()
	ownerAsGuest.Guests = []domain.UserID{"u-owner"}
	if _, err := domain.MeetingFromSnapshot(ownerAsGuest); !errors.Is(err, domain.ErrInvalidSnapshot) {
		t.Errorf("owner in guests: want ErrInvalidSnapshot, got %v", err)
	}

	noOwner := m.Snapshot()
	noOwner.OwnerID = ""
	if _, err := domain.MeetingFromSnapshot(noOwner); !errors.Is(err, domain.ErrInvalidSnapshot) {
		t.Errorf("empty owner: want ErrInvalidSnapshot, got %v", err)
	}

	badDuration := m.Snapshot()
	badDuration.Duration = 90 * time.Second
	if _, err := domain.MeetingFromSnapshot(badDuration); !errors.Is(err, domain.ErrInvalidSnapshot) {
		t.Errorf("bad duration: want ErrInvalidSnapshot, got %v", err)
	}
}
