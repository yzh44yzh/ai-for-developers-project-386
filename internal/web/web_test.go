package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/yzh44yzh/bookmeet/internal/domain"
	"github.com/yzh44yzh/bookmeet/internal/web"
)

// fakeUserStore is an in-memory domain.UserStore for handler tests.
type fakeUserStore struct {
	byID map[domain.UserID]domain.User
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{byID: make(map[domain.UserID]domain.User)}
}

func (f *fakeUserStore) Add(_ context.Context, u domain.User) error {
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

func (f *fakeMeetingStore) ListByOwner(ctx context.Context, ownerID domain.UserID) ([]domain.Meeting, error) {
	all, err := f.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Meeting, 0, len(all))
	for _, m := range all {
		if m.OwnerID() == ownerID {
			out = append(out, m)
		}
	}
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

var (
	alice = domain.User{ID: domain.NewUserID(), FirstName: "Alice", LastName: "Anderson", Email: "alice@bookmeet.dev"}
	bob   = domain.User{ID: domain.NewUserID(), FirstName: "Bob", LastName: "Brown", Email: "bob@bookmeet.dev"}
	carol = domain.User{ID: domain.NewUserID(), FirstName: "Carol", LastName: "Clark", Email: "carol@bookmeet.dev"}
)

func newStores() (*fakeUserStore, *fakeMeetingStore) {
	users := newFakeUserStore()
	for _, u := range []domain.User{alice, bob, carol} {
		_ = users.Add(context.Background(), u)
	}
	return users, newFakeMeetingStore()
}

func newMux() *http.ServeMux {
	users, meetings := newStores()
	return web.NewMux(users, meetings)
}

// mustMeeting builds a valid Meeting with the given details.
func mustMeeting(t *testing.T, ownerID domain.UserID, title string, start time.Time, dur time.Duration) domain.Meeting {
	t.Helper()
	m, err := domain.NewMeeting(domain.NewMeetingID(), ownerID, title, start, dur, "description of "+title, time.Now())
	if err != nil {
		t.Fatalf("NewMeeting: %v", err)
	}
	return m
}

// login POSTs the email to /login and returns the recorder.
func login(t *testing.T, mux http.Handler, email string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"email": {email}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)
	return rw
}

// sessionCookie logs the user in and returns the session cookie.
func sessionCookie(t *testing.T, mux http.Handler, email string) *http.Cookie {
	t.Helper()
	rw := login(t, mux, email)
	cookies := rw.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie from login, got %d", len(cookies))
	}
	return cookies[0]
}

// get performs GET path with the cookie and returns the recorder.
func get(t *testing.T, mux http.Handler, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)
	return rw
}

func TestLoginFormRenders(t *testing.T) {
	rw := get(t, newMux(), "/login", nil)

	if rw.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusOK)
	}
	body := rw.Body.String()
	if !strings.Contains(body, `name="email"`) || !strings.Contains(body, "<form") {
		t.Fatalf("login form not rendered: %s", body)
	}
}

func TestLoginUnknownEmail(t *testing.T) {
	rw := login(t, newMux(), "nobody@bookmeet.dev")

	if rw.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusOK)
	}
	if !strings.Contains(rw.Body.String(), "user not exists") {
		t.Fatalf("body: got %q, want it to contain %q", rw.Body.String(), "user not exists")
	}
	if cookies := rw.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("expected no cookies, got %v", cookies)
	}
}

func TestLoginSuccessSetsSessionAndRedirects(t *testing.T) {
	rw := login(t, newMux(), alice.Email)
	res := rw.Result()

	if rw.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusSeeOther)
	}
	if loc := res.Header.Get("Location"); loc != "/meetings" {
		t.Fatalf("Location: got %q, want %q", loc, "/meetings")
	}
	cookies := res.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != "bookmeet_session" || c.Value == "" || !c.HttpOnly || c.Path != "/" {
		t.Fatalf("unexpected cookie: %+v", c)
	}
}

func TestHomeWithSessionGreetsUser(t *testing.T) {
	mux := newMux()
	rw := get(t, mux, "/", sessionCookie(t, mux, alice.Email))

	if rw.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusOK)
	}
	if !strings.Contains(rw.Body.String(), "Hello Alice Anderson") {
		t.Fatalf("body: got %q, want it to contain %q", rw.Body.String(), "Hello Alice Anderson")
	}
	if !strings.Contains(rw.Body.String(), `href="/meetings"`) {
		t.Fatalf("body: got %q, want a link to /meetings", rw.Body.String())
	}
}

func TestHomeWithoutSessionRedirectsToLogin(t *testing.T) {
	rw := get(t, newMux(), "/", nil)

	if rw.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusSeeOther)
	}
	if loc := rw.Result().Header.Get("Location"); loc != "/login" {
		t.Fatalf("Location: got %q, want %q", loc, "/login")
	}
}

func TestHomeWithBogusCookieRedirectsToLogin(t *testing.T) {
	rw := get(t, newMux(), "/", &http.Cookie{Name: "bookmeet_session", Value: "bogus"})

	if rw.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusSeeOther)
	}
	if loc := rw.Result().Header.Get("Location"); loc != "/login" {
		t.Fatalf("Location: got %q, want %q", loc, "/login")
	}
}

func TestLoginFormWhenLoggedInRedirectsHome(t *testing.T) {
	mux := newMux()
	rw := get(t, mux, "/login", sessionCookie(t, mux, alice.Email))

	if rw.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusSeeOther)
	}
	if loc := rw.Result().Header.Get("Location"); loc != "/" {
		t.Fatalf("Location: got %q, want %q", loc, "/")
	}
}

func TestMeetingsRequiresLogin(t *testing.T) {
	rw := get(t, newMux(), "/meetings", nil)

	if rw.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusSeeOther)
	}
	if loc := rw.Result().Header.Get("Location"); loc != "/login" {
		t.Fatalf("Location: got %q, want %q", loc, "/login")
	}
}

func TestMeetingsListsOwnedUpcomingWithDetails(t *testing.T) {
	users, meetings := newStores()
	mux := web.NewMux(users, meetings)
	ctx := context.Background()
	now := time.Now()

	alpha := mustMeeting(t, alice.ID, "Alpha", now.Add(time.Hour), 30*time.Minute)
	if err := alpha.AddGuest(bob.ID, now); err != nil {
		t.Fatalf("AddGuest: %v", err)
	}
	beta := mustMeeting(t, alice.ID, "Beta", now.Add(2*time.Hour), 45*time.Minute)
	gamma := mustMeeting(t, alice.ID, "Gamma", now.Add(3*time.Hour), time.Hour)
	if err := gamma.Cancel(now); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	past, err := domain.MeetingFromSnapshot(domain.MeetingSnapshot{
		ID: domain.NewMeetingID(), OwnerID: alice.ID, Title: "Ancient",
		Start: now.Add(-2 * time.Hour), Duration: 30 * time.Minute, Description: "long over",
	})
	if err != nil {
		t.Fatalf("MeetingFromSnapshot: %v", err)
	}
	bobs := mustMeeting(t, bob.ID, "BobsMeeting", now.Add(time.Hour), 30*time.Minute)

	for _, m := range []domain.Meeting{alpha, beta, gamma, past, bobs} {
		if err := meetings.Create(ctx, m); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	rw := get(t, mux, "/meetings", sessionCookie(t, mux, alice.Email))

	if rw.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusOK)
	}
	body := rw.Body.String()

	for _, want := range []string{
		"Alpha", "Beta", "Gamma",
		"description of Alpha", "30 min", "45 min",
		"scheduled", "draft", "cancelled", "Cancelled at",
		"Bob Brown", "bob@bookmeet.dev",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
	for _, unwanted := range []string{"Ancient", "BobsMeeting"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("body should not contain %q", unwanted)
		}
	}
	if !strings.Contains(body, "<dd>none</dd>") {
		t.Errorf("guestless meetings should show no guests: %s", body)
	}
	if strings.Index(body, "Alpha") > strings.Index(body, "Beta") ||
		strings.Index(body, "Beta") > strings.Index(body, "Gamma") {
		t.Errorf("meetings not ordered by start ascending: %s", body)
	}
}

func TestMeetingsEmptyState(t *testing.T) {
	mux := newMux()
	rw := get(t, mux, "/meetings", sessionCookie(t, mux, carol.Email))

	if rw.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusOK)
	}
	if !strings.Contains(rw.Body.String(), "You have no meetings.") {
		t.Fatalf("body: got %q, want it to contain %q", rw.Body.String(), "You have no meetings.")
	}
}
