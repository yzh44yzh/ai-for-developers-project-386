package domain_test

import (
	"errors"
	"testing"

	"github.com/yzh44yzh/bookmeet/internal/domain"
)

func TestUsersAddAndGet(t *testing.T) {
	users := domain.NewUsers()

	user, err := users.Add("u1", "Ada", "Lovelace", "First programmer", "ada@example.com")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := users.Get(user.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.FirstName != "Ada" || got.LastName != "Lovelace" || got.Email != "ada@example.com" {
		t.Errorf("got %+v", got)
	}
}

func TestUsersRejectsDuplicateEmail(t *testing.T) {
	users := domain.NewUsers()
	if _, err := users.Add("u1", "Ada", "Lovelace", "", "ada@example.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, err := users.Add("u2", "Grace", "Hopper", "", "ada@example.com")
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("want ErrEmailTaken, got %v", err)
	}
}

func TestUsersRejectsDuplicateID(t *testing.T) {
	users := domain.NewUsers()
	if _, err := users.Add("u1", "Ada", "Lovelace", "", "ada@example.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, err := users.Add("u1", "Grace", "Hopper", "", "grace@example.com")
	if !errors.Is(err, domain.ErrUserIDTaken) {
		t.Fatalf("want ErrUserIDTaken, got %v", err)
	}

	// The original User is untouched and the email index is consistent.
	got, err := users.Get("u1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.FirstName != "Ada" {
		t.Errorf("FirstName = %q, want Ada", got.FirstName)
	}
}
