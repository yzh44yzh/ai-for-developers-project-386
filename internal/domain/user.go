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

// Users is the store of all registered Users.
type Users struct {
	byID    map[UserID]User
	byEmail map[string]UserID
}

func NewUsers() *Users {
	return &Users{
		byID:    make(map[UserID]User),
		byEmail: make(map[string]UserID),
	}
}

// Add registers a new User. Both the ID and the email must be unique.
func (u *Users) Add(id UserID, firstName, lastName, description, email string) (User, error) {
	if _, taken := u.byID[id]; taken {
		return User{}, ErrUserIDTaken
	}
	if _, taken := u.byEmail[email]; taken {
		return User{}, ErrEmailTaken
	}
	user := User{
		ID:          id,
		FirstName:   firstName,
		LastName:    lastName,
		Description: description,
		Email:       email,
	}
	u.byID[id] = user
	u.byEmail[email] = id
	return user, nil
}

// Get returns the User with the given ID.
func (u *Users) Get(id UserID) (User, error) {
	user, ok := u.byID[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}
