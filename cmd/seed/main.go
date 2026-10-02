// Command seed fills the BookMeet database with demo data: Users from a
// fixed roster, each owning randomly generated Meetings. Users that already
// exist (by email) are skipped together with their Meetings, so rerunning
// the command on a seeded database is a no-op.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yzh44yzh/bookmeet/internal/domain"
	"github.com/yzh44yzh/bookmeet/internal/postgres"
)

// person is a roster entry; the User ID is generated per run.
type person struct {
	firstName   string
	lastName    string
	description string
	email       string
}

var roster = []person{
	{"Alice", "Anderson", "Backend engineer", "alice@bookmeet.dev"},
	{"Bob", "Brown", "Product manager", "bob@bookmeet.dev"},
	{"Carol", "Clark", "Designer", "carol@bookmeet.dev"},
	{"Dave", "Davis", "QA engineer", "dave@bookmeet.dev"},
	{"Erin", "Evans", "Frontend engineer", "erin@bookmeet.dev"},
	{"Frank", "Foster", "DevOps engineer", "frank@bookmeet.dev"},
	{"Grace", "Green", "Engineering manager", "grace@bookmeet.dev"},
	{"Heidi", "Hill", "Data analyst", "heidi@bookmeet.dev"},
	{"Ivan", "Irwin", "Security engineer", "ivan@bookmeet.dev"},
	{"Judy", "Jones", "Technical writer", "judy@bookmeet.dev"},
}

var meetingTitles = []string{
	"Weekly sync",
	"1:1 catch-up",
	"Sprint planning",
	"Design review",
	"Retrospective",
	"Standup",
	"Roadmap discussion",
	"Interview debrief",
	"Demo",
	"Brainstorm",
}

var meetingDescriptions = []string{
	"Auto-generated demo meeting.",
	"Seeded for local development.",
	"Random demo data, safe to delete.",
}

var meetingDurations = []time.Duration{30 * time.Minute, 60 * time.Minute, 90 * time.Minute}

func main() {
	numUsers := flag.Int("users", 2, "number of users to seed")
	numMeetings := flag.Int("meetings", 3, "number of meetings per user")
	flag.Parse()

	if *numUsers < 1 || *numUsers > len(roster) {
		log.Fatalf("users must be between 1 and %d", len(roster))
	}
	if *numMeetings < 1 {
		log.Fatal("meetings must be at least 1")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Local development default; dev-only credentials.
		dsn = "postgres://test:test@localhost:5432/testdb?sslmode=disable"
	}
	ctx := context.Background()

	if err := postgres.Migrate(dsn); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	users := postgres.NewUsers(pool)
	meetings := postgres.NewMeetings(pool)

	created := make([]domain.User, 0, *numUsers)
	for _, p := range roster[:*numUsers] {
		u := domain.User{
			ID:          domain.NewUserID(),
			FirstName:   p.firstName,
			LastName:    p.lastName,
			Description: p.description,
			Email:       p.email,
		}
		switch err := users.Add(ctx, u); {
		case err == nil:
			log.Printf("created user %s %s <%s>", u.FirstName, u.LastName, u.Email)
			created = append(created, u)
		case errors.Is(err, domain.ErrEmailTaken):
			log.Printf("user %s exists, skipping", u.Email)
		default:
			log.Fatalf("add user %s: %v", u.Email, err)
		}
	}

	now := time.Now()
	for _, owner := range created {
		for range *numMeetings {
			m, err := randomMeeting(owner.ID, now)
			if err != nil {
				log.Fatalf("build meeting: %v", err)
			}
			if len(created) > 1 && rand.Intn(2) == 0 {
				guest := randomOther(created, owner.ID)
				if err := m.AddGuest(guest.ID, now); err != nil {
					log.Fatalf("add guest: %v", err)
				}
			}
			if err := meetings.Create(ctx, m); err != nil {
				log.Fatalf("create meeting: %v", err)
			}
			log.Printf("created meeting %q for %s (%s, starts %s, %v)",
				m.Title(), owner.Email, m.Status(), m.Start().Format(time.RFC3339), m.Duration())
		}
	}
}

// randomMeeting builds a Meeting starting 1–30 days from now with a random
// title, duration and description.
func randomMeeting(ownerID domain.UserID, now time.Time) (domain.Meeting, error) {
	start := now.Add(time.Duration(24+rand.Intn(30*24)) * time.Hour).Truncate(time.Minute)
	return domain.NewMeeting(
		domain.NewMeetingID(),
		ownerID,
		meetingTitles[rand.Intn(len(meetingTitles))],
		start,
		meetingDurations[rand.Intn(len(meetingDurations))],
		meetingDescriptions[rand.Intn(len(meetingDescriptions))],
		now,
	)
}

// randomOther picks a random user from users whose ID differs from except.
func randomOther(users []domain.User, except domain.UserID) domain.User {
	for {
		if u := users[rand.Intn(len(users))]; u.ID != except {
			return u
		}
	}
}
