package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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

var alice = domain.User{
	ID:        domain.NewUserID(),
	FirstName: "Alice",
	LastName:  "Anderson",
	Email:     "alice@bookmeet.dev",
}

func newMux() *http.ServeMux {
	users := newFakeUserStore()
	_ = users.Add(context.Background(), alice)
	return web.NewMux(users)
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

// sessionCookie logs alice in and returns her session cookie.
func sessionCookie(t *testing.T, mux http.Handler) *http.Cookie {
	t.Helper()
	rw := login(t, mux, alice.Email)
	cookies := rw.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie from login, got %d", len(cookies))
	}
	return cookies[0]
}

func TestLoginFormRenders(t *testing.T) {
	rw := httptest.NewRecorder()
	newMux().ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/login", nil))

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
	if loc := res.Header.Get("Location"); loc != "/" {
		t.Fatalf("Location: got %q, want %q", loc, "/")
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
	cookie := sessionCookie(t, mux)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusOK)
	}
	if !strings.Contains(rw.Body.String(), "Hello Alice Anderson") {
		t.Fatalf("body: got %q, want it to contain %q", rw.Body.String(), "Hello Alice Anderson")
	}
}

func TestHomeWithoutSessionRedirectsToLogin(t *testing.T) {
	rw := httptest.NewRecorder()
	newMux().ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/", nil))

	if rw.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusSeeOther)
	}
	if loc := rw.Result().Header.Get("Location"); loc != "/login" {
		t.Fatalf("Location: got %q, want %q", loc, "/login")
	}
}

func TestHomeWithBogusCookieRedirectsToLogin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "bookmeet_session", Value: "bogus"})
	rw := httptest.NewRecorder()
	newMux().ServeHTTP(rw, req)

	if rw.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusSeeOther)
	}
	if loc := rw.Result().Header.Get("Location"); loc != "/login" {
		t.Fatalf("Location: got %q, want %q", loc, "/login")
	}
}

func TestLoginFormWhenLoggedInRedirectsHome(t *testing.T) {
	mux := newMux()
	cookie := sessionCookie(t, mux)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.AddCookie(cookie)
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want %d", rw.Code, http.StatusSeeOther)
	}
	if loc := rw.Result().Header.Get("Location"); loc != "/" {
		t.Fatalf("Location: got %q, want %q", loc, "/")
	}
}
