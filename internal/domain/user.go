package domain

import "errors"

var (
	ErrEmailTaken   = errors.New("email already taken")
	ErrUserIDTaken  = errors.New("user ID already taken")
	ErrUserNotFound = errors.New("user not found")
)

type UserID string

// User is a person registered in BookMeet.
type User struct {
	ID          UserID
	FirstName   string
	LastName    string
	Description string
	Email       string
}
