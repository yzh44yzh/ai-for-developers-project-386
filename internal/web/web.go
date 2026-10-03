// Package web serves the BookMeet HTML pages: email login backed by in-memory
// sessions. See docs/adr/0003-email-login.md.
package web

import (
	"embed"
	"errors"
	"html/template"
	"log"
	"net/http"

	"github.com/yzh44yzh/bookmeet/internal/domain"
)

//go:embed templates/*.html
var templatesFS embed.FS

var templates = template.Must(template.ParseFS(templatesFS, "templates/*.html"))

const sessionCookie = "bookmeet_session"

// NewMux routes the BookMeet HTML pages backed by the given user store.
func NewMux(users domain.UserStore) *http.ServeMux {
	w := &web{users: users, sessions: newSessions()}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", w.loginForm)
	mux.HandleFunc("POST /login", w.login)
	mux.HandleFunc("GET /{$}", w.home)
	return mux
}

type web struct {
	users    domain.UserStore
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
		http.Redirect(rw, r, "/", http.StatusSeeOther)
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
