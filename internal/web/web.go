// Package web serves the BookMeet HTML pages: email login backed by in-memory
// sessions. See docs/adr/0003-email-login.md.
package web

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/yzh44yzh/bookmeet/internal/domain"
)

//go:embed templates/*.html
var templatesFS embed.FS

var templates = template.Must(template.ParseFS(templatesFS, "templates/*.html"))

const sessionCookie = "bookmeet_session"

// NewMux routes the BookMeet HTML pages backed by the given stores.
func NewMux(users domain.UserStore, meetings domain.MeetingStore) *http.ServeMux {
	w := &web{users: users, meetings: meetings, sessions: newSessions()}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", w.loginForm)
	mux.HandleFunc("POST /login", w.login)
	mux.HandleFunc("GET /{$}", w.home)
	mux.HandleFunc("GET /meetings", w.myMeetings)
	mux.HandleFunc("GET /meetings/new", w.newMeetingForm)
	mux.HandleFunc("POST /meetings/new", w.createMeeting)
	return mux
}

type web struct {
	users    domain.UserStore
	meetings domain.MeetingStore
	sessions *sessions
}

func (w *web) loginForm(rw http.ResponseWriter, r *http.Request) {
	if _, ok := w.loggedInUser(r); ok {
		http.Redirect(rw, r, "/", http.StatusSeeOther)
		return
	}
	render(rw, "login.html", nil)
}

func (w *web) login(rw http.ResponseWriter, r *http.Request) {
	u, err := w.users.GetByEmail(r.Context(), r.PostFormValue("email"))
	switch {
	case err == nil:
		http.SetCookie(rw, &http.Cookie{
			Name:     sessionCookie,
			Value:    w.sessions.create(u.ID),
			Path:     "/",
			HttpOnly: true,
		})
		http.Redirect(rw, r, "/meetings", http.StatusSeeOther)
	case errors.Is(err, domain.ErrUserNotFound):
		render(rw, "not_found.html", nil)
	default:
		http.Error(rw, "internal error", http.StatusInternalServerError)
	}
}

func (w *web) home(rw http.ResponseWriter, r *http.Request) {
	u, ok := w.loggedInUser(r)
	if !ok {
		http.Redirect(rw, r, "/login", http.StatusSeeOther)
		return
	}
	render(rw, "hello.html", u)
}

// myMeetings lists the logged-in User's upcoming Meetings with all details.
// Past (frozen) Meetings are not shown.
func (w *web) myMeetings(rw http.ResponseWriter, r *http.Request) {
	u, ok := w.loggedInUser(r)
	if !ok {
		http.Redirect(rw, r, "/login", http.StatusSeeOther)
		return
	}
	meetings, err := w.meetings.ListByOwner(r.Context(), u.ID)
	if err != nil {
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	views := make([]meetingView, 0, len(meetings))
	for _, m := range meetings {
		if m.Frozen(now) {
			continue
		}
		views = append(views, w.toView(r, m))
	}
	render(rw, "meetings.html", views)
}

// meetingFormView renders the create-meeting form.
type meetingFormView struct {
	Error string
}

func (w *web) newMeetingForm(rw http.ResponseWriter, r *http.Request) {
	if _, ok := w.loggedInUser(r); !ok {
		http.Redirect(rw, r, "/login", http.StatusSeeOther)
		return
	}
	render(rw, "meeting_new.html", meetingFormView{})
}

// createMeeting creates a Meeting owned by the logged-in User. On success it
// redirects to /meetings; on invalid input it re-renders the form with the
// error (see docs/adr/0005-create-meeting-form.md).
func (w *web) createMeeting(rw http.ResponseWriter, r *http.Request) {
	u, ok := w.loggedInUser(r)
	if !ok {
		http.Redirect(rw, r, "/login", http.StatusSeeOther)
		return
	}

	fail := func(msg string) {
		rw.WriteHeader(http.StatusBadRequest)
		render(rw, "meeting_new.html", meetingFormView{Error: msg})
	}

	title := r.PostFormValue("title")
	if title == "" {
		fail("title is required")
		return
	}
	start, err := parseLocalStart(r.PostFormValue("start"))
	if err != nil {
		fail("invalid start")
		return
	}
	mins, err := strconv.Atoi(r.PostFormValue("duration"))
	if err != nil {
		fail("invalid duration")
		return
	}

	m, err := domain.NewMeeting(
		domain.NewMeetingID(), u.ID, title, start,
		time.Duration(mins)*time.Minute, r.PostFormValue("description"), time.Now())
	if err != nil {
		fail(err.Error())
		return
	}
	if err := w.meetings.Create(r.Context(), m); err != nil {
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(rw, r, "/meetings", http.StatusSeeOther)
}

// parseLocalStart parses a datetime-local value ("2006-01-02T15:04") in the
// server's local timezone; the browser sends no zone (see ADR 0005).
func parseLocalStart(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid datetime-local value")
}

// meetingView is a Meeting rendered on the my-meetings page.
type meetingView struct {
	Title       string
	Start       string
	Duration    string
	Description string
	Status      string
	CancelledAt string // empty unless Cancelled
	Guests      []string
}

func (w *web) toView(r *http.Request, m domain.Meeting) meetingView {
	v := meetingView{
		Title:       m.Title(),
		Start:       m.Start().Format("2006-01-02 15:04 MST"),
		Duration:    fmt.Sprintf("%d min", m.Duration()/time.Minute),
		Description: m.Description(),
		Status:      string(m.Status()),
	}
	if cancelledAt := m.CancelledAt(); cancelledAt != nil {
		v.CancelledAt = cancelledAt.Format("2006-01-02 15:04 MST")
	}
	for _, id := range m.Guests() {
		guest, err := w.users.Get(r.Context(), id)
		if err != nil {
			v.Guests = append(v.Guests, string(id)) // unresolvable: show the raw ID
			continue
		}
		v.Guests = append(v.Guests, fmt.Sprintf("%s %s <%s>", guest.FirstName, guest.LastName, guest.Email))
	}
	return v
}

// loggedInUser resolves the request's session cookie to a User.
func (w *web) loggedInUser(r *http.Request) (domain.User, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return domain.User{}, false
	}
	id, ok := w.sessions.get(cookie.Value)
	if !ok {
		return domain.User{}, false
	}
	u, err := w.users.Get(r.Context(), id)
	if err != nil {
		return domain.User{}, false
	}
	return u, true
}

func render(rw http.ResponseWriter, name string, data any) {
	if err := templates.ExecuteTemplate(rw, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}
