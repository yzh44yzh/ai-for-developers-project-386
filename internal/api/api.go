// Package api exposes the BookMeet JSON HTTP API over the domain stores.
package api

import (
	"net/http"

	"github.com/yzh44yzh/bookmeet/internal/domain"
)

// NewMux routes the BookMeet HTTP API to handlers backed by the given stores.
func NewMux(users domain.UserStore, meetings domain.MeetingStore) *http.ServeMux {
	a := &api{users: users, meetings: meetings}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", a.createUser)
	mux.HandleFunc("GET /users/{id}", a.getUser)
	mux.HandleFunc("GET /users", a.getUserByEmail)
	mux.HandleFunc("POST /meetings", a.createMeeting)
	mux.HandleFunc("GET /meetings", a.listMeetings)
	mux.HandleFunc("GET /meetings/{id}", a.getMeeting)
	mux.HandleFunc("PATCH /meetings/{id}", a.editMeeting)
	mux.HandleFunc("POST /meetings/{id}/cancel", a.cancelMeeting)
	mux.HandleFunc("POST /meetings/{id}/guests", a.addGuest)
	mux.HandleFunc("DELETE /meetings/{id}/guests/{user_id}", a.removeGuest)
	return mux
}

type api struct {
	users    domain.UserStore
	meetings domain.MeetingStore
}
