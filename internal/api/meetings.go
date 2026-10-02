package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/yzh44yzh/bookmeet/internal/domain"
)

type meetingJSON struct {
	ID              domain.MeetingID `json:"id"`
	OwnerID         domain.UserID    `json:"owner_id"`
	Title           string           `json:"title"`
	Start           time.Time        `json:"start"`
	DurationMinutes int              `json:"duration_minutes"`
	Description     string           `json:"description"`
	Guests          []domain.UserID  `json:"guests"`
	Status          domain.Status    `json:"status"`
	CancelledAt     *time.Time       `json:"cancelled_at"`
}

func toMeetingJSON(m domain.Meeting) meetingJSON {
	return meetingJSON{
		ID:              m.ID(),
		OwnerID:         m.OwnerID(),
		Title:           m.Title(),
		Start:           m.Start(),
		DurationMinutes: int(m.Duration() / time.Minute),
		Description:     m.Description(),
		Guests:          m.Guests(),
		Status:          m.Status(),
		CancelledAt:     m.CancelledAt(),
	}
}

type createMeetingRequest struct {
	OwnerID         string    `json:"owner_id"`
	Title           string    `json:"title"`
	Start           time.Time `json:"start"`
	DurationMinutes int       `json:"duration_minutes"`
	Description     string    `json:"description"`
}

func (a *api) createMeeting(w http.ResponseWriter, r *http.Request) {
	var req createMeetingRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !parseUUID(w, "owner_id", req.OwnerID) {
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.Start.IsZero() {
		writeError(w, http.StatusBadRequest, "start is required")
		return
	}
	if req.DurationMinutes <= 0 {
		writeError(w, http.StatusBadRequest, "duration_minutes must be positive")
		return
	}
	if _, err := a.users.Get(r.Context(), domain.UserID(req.OwnerID)); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			writeError(w, http.StatusBadRequest, "unknown owner_id")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	m, err := domain.NewMeeting(
		domain.NewMeetingID(),
		domain.UserID(req.OwnerID),
		req.Title,
		req.Start,
		time.Duration(req.DurationMinutes)*time.Minute,
		req.Description,
		time.Now(),
	)
	if errors.Is(err, domain.ErrStartInPast) {
		writeError(w, http.StatusBadRequest, "start must be in the future")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := a.meetings.Create(r.Context(), m); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, toMeetingJSON(m))
}

func (a *api) listMeetings(w http.ResponseWriter, r *http.Request) {
	meetings, err := a.meetings.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	out := make([]meetingJSON, 0, len(meetings))
	for _, m := range meetings {
		out = append(out, toMeetingJSON(m))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *api) getMeeting(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !parseUUID(w, "id", id) {
		return
	}
	m, err := a.meetings.Get(r.Context(), domain.MeetingID(id))
	if errors.Is(err, domain.ErrMeetingNotFound) {
		writeError(w, http.StatusNotFound, "meeting not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toMeetingJSON(m))
}

type editMeetingRequest struct {
	Title           string    `json:"title"`
	Start           time.Time `json:"start"`
	DurationMinutes int       `json:"duration_minutes"`
	Description     string    `json:"description"`
}

func (a *api) editMeeting(w http.ResponseWriter, r *http.Request) {
	var req editMeetingRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Title == "" || req.Start.IsZero() || req.DurationMinutes <= 0 {
		writeError(w, http.StatusBadRequest, "title, start and positive duration_minutes are required")
		return
	}
	now := time.Now()
	a.updateMeeting(w, r, func(m *domain.Meeting) error {
		return m.EditDetails(req.Title, req.Start, time.Duration(req.DurationMinutes)*time.Minute, req.Description, now)
	})
}

func (a *api) cancelMeeting(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	a.updateMeeting(w, r, func(m *domain.Meeting) error {
		return m.Cancel(now)
	})
}

type addGuestRequest struct {
	UserID string `json:"user_id"`
}

func (a *api) addGuest(w http.ResponseWriter, r *http.Request) {
	var req addGuestRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !parseUUID(w, "user_id", req.UserID) {
		return
	}
	if _, err := a.users.Get(r.Context(), domain.UserID(req.UserID)); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			writeError(w, http.StatusBadRequest, "unknown user_id")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	now := time.Now()
	a.updateMeeting(w, r, func(m *domain.Meeting) error {
		return m.AddGuest(domain.UserID(req.UserID), now)
	})
}

func (a *api) removeGuest(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("user_id")
	if !parseUUID(w, "user_id", uid) {
		return
	}
	now := time.Now()
	a.updateMeeting(w, r, func(m *domain.Meeting) error {
		return m.RemoveGuest(domain.UserID(uid), now)
	})
}

// updateMeeting runs fn against the Meeting identified by the {id} path
// value and maps domain mutation errors to HTTP statuses. Success is
// 204 No Content.
func (a *api) updateMeeting(w http.ResponseWriter, r *http.Request, fn func(m *domain.Meeting) error) {
	id := r.PathValue("id")
	if !parseUUID(w, "id", id) {
		return
	}
	err := a.meetings.Update(r.Context(), domain.MeetingID(id), fn)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, domain.ErrMeetingNotFound):
		writeError(w, http.StatusNotFound, "meeting not found")
	case errors.Is(err, domain.ErrMeetingCancelled):
		writeError(w, http.StatusConflict, "meeting is cancelled")
	case errors.Is(err, domain.ErrMeetingFrozen):
		writeError(w, http.StatusConflict, "meeting is frozen")
	case errors.Is(err, domain.ErrDuplicateGuest):
		writeError(w, http.StatusConflict, "user is already a guest")
	case errors.Is(err, domain.ErrOwnerAsGuest):
		writeError(w, http.StatusBadRequest, "owner cannot be a guest")
	case errors.Is(err, domain.ErrNotGuest):
		writeError(w, http.StatusBadRequest, "user is not a guest")
	case errors.Is(err, domain.ErrInvalidDuration):
		writeError(w, http.StatusBadRequest, "duration must be a positive whole number of minutes")
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
