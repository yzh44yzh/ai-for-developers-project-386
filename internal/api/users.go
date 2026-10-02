package api

import (
	"errors"
	"net/http"

	"github.com/yzh44yzh/bookmeet/internal/domain"
)

type userJSON struct {
	ID          domain.UserID `json:"id"`
	FirstName   string        `json:"first_name"`
	LastName    string        `json:"last_name"`
	Description string        `json:"description"`
	Email       string        `json:"email"`
}

func toUserJSON(u domain.User) userJSON {
	return userJSON{
		ID:          u.ID,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		Description: u.Description,
		Email:       u.Email,
	}
}

type createUserRequest struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Description string `json:"description"`
	Email       string `json:"email"`
}

func (a *api) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.FirstName == "" || req.LastName == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "first_name, last_name and email are required")
		return
	}

	u := domain.User{
		ID:          domain.NewUserID(),
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Description: req.Description,
		Email:       req.Email,
	}
	err := a.users.Add(r.Context(), u)
	switch {
	case err == nil:
		writeJSON(w, http.StatusCreated, toUserJSON(u))
	case errors.Is(err, domain.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email already taken")
	case errors.Is(err, domain.ErrUserIDTaken):
		writeError(w, http.StatusConflict, "user ID already taken")
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func (a *api) getUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !parseUUID(w, "id", id) {
		return
	}
	u, err := a.users.Get(r.Context(), domain.UserID(id))
	if err != nil {
		a.userError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserJSON(u))
}

func (a *api) getUserByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		writeError(w, http.StatusBadRequest, "email query parameter is required")
		return
	}
	u, err := a.users.GetByEmail(r.Context(), email)
	if err != nil {
		a.userError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserJSON(u))
}

func (a *api) userError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrUserNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeError(w, http.StatusInternalServerError, "internal error")
}
