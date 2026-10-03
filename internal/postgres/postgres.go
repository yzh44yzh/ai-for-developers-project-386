// Package postgres implements the domain store interfaces (UserStore,
// MeetingStore) on PostgreSQL.
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/yzh44yzh/bookmeet/internal/domain"
)

//go:embed migrations/*.sql
var migrations embed.FS

// uniqueViolation is the PostgreSQL SQLSTATE for unique constraint breaches.
const uniqueViolation = "23505"

// Migrate applies all pending schema migrations.
func Migrate(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, "migrations")
}

// Users implements domain.UserStore.
type Users struct {
	pool *pgxpool.Pool
}

// Meetings implements domain.MeetingStore.
type Meetings struct {
	pool *pgxpool.Pool
}

func NewUsers(pool *pgxpool.Pool) *Users       { return &Users{pool: pool} }
func NewMeetings(pool *pgxpool.Pool) *Meetings { return &Meetings{pool: pool} }

var (
	_ domain.UserStore    = (*Users)(nil)
	_ domain.MeetingStore = (*Meetings)(nil)
)

// Add registers a new User. Both the ID and the email must be unique.
func (s *Users) Add(ctx context.Context, u domain.User) error {
	_, err := s.pool.Exec(ctx,
		"INSERT INTO users (id, first_name, last_name, description, email) VALUES ($1::uuid, $2, $3, $4, $5)",
		u.ID, u.FirstName, u.LastName, u.Description, u.Email)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			if pgErr.ConstraintName == "users_email_key" {
				return domain.ErrEmailTaken
			}
			return domain.ErrUserIDTaken
		}
		return fmt.Errorf("add user: %w", err)
	}
	return nil
}

// Get returns the User with the given ID.
func (s *Users) Get(ctx context.Context, id domain.UserID) (domain.User, error) {
	var u domain.User
	err := s.pool.QueryRow(ctx,
		"SELECT id::text, first_name, last_name, description, email FROM users WHERE id = $1::uuid", id).
		Scan(&u.ID, &u.FirstName, &u.LastName, &u.Description, &u.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

// GetByEmail returns the User with the given email.
func (s *Users) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	var u domain.User
	err := s.pool.QueryRow(ctx,
		"SELECT id::text, first_name, last_name, description, email FROM users WHERE email = $1", email).
		Scan(&u.ID, &u.FirstName, &u.LastName, &u.Description, &u.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

// meetingColumns lists the meetings columns for INSERT; meetingSelect lists
// them for SELECT, casting uuid columns to text so they scan into the
// domain's string ID types.
const (
	meetingColumns = "id, owner_id, title, start, duration_minutes, description, cancelled_at"
	meetingSelect  = "id::text, owner_id::text, title, start, duration_minutes, description, cancelled_at"
)

// Create persists a new Meeting, including any Guests it already carries.
func (s *Meetings) Create(ctx context.Context, m domain.Meeting) error {
	snapshot := m.Snapshot()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		"INSERT INTO meetings ("+meetingColumns+") VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)",
		snapshot.ID, snapshot.OwnerID, snapshot.Title, snapshot.Start, minutes(snapshot.Duration), snapshot.Description, snapshot.CancelledAt); err != nil {
		return fmt.Errorf("create meeting: %w", err)
	}
	for _, g := range snapshot.Guests {
		if _, err := tx.Exec(ctx,
			"INSERT INTO meeting_guests (meeting_id, user_id) VALUES ($1::uuid, $2::uuid)", snapshot.ID, g); err != nil {
			return fmt.Errorf("add guest: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// Get returns the Meeting with the given ID.
func (s *Meetings) Get(ctx context.Context, id domain.MeetingID) (domain.Meeting, error) {
	snapshot, err := s.scanMeeting(ctx, s.pool, id, false)
	if err != nil {
		return domain.Meeting{}, err
	}
	return reconstitute(id, snapshot)
}

// List returns all Meetings.
func (s *Meetings) List(ctx context.Context) ([]domain.Meeting, error) {
	return s.queryMeetings(ctx, "SELECT "+meetingSelect+" FROM meetings ORDER BY start, id")
}

// ListByOwner returns all Meetings owned by the given User.
func (s *Meetings) ListByOwner(ctx context.Context, owner domain.UserID) ([]domain.Meeting, error) {
	return s.queryMeetings(ctx,
		"SELECT "+meetingSelect+" FROM meetings WHERE owner_id = $1::uuid ORDER BY start, id", owner)
}

func (s *Meetings) queryMeetings(ctx context.Context, query string, args ...any) ([]domain.Meeting, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list meetings: %w", err)
	}
	defer rows.Close()

	var meetings []domain.Meeting
	for rows.Next() {
		snapshot, err := scanMeetingRow(rows)
		if err != nil {
			return nil, err
		}
		snapshot.Guests, err = s.guestIDs(ctx, s.pool, snapshot.ID)
		if err != nil {
			return nil, err
		}
		m, err := reconstitute(snapshot.ID, snapshot)
		if err != nil {
			return nil, err
		}
		meetings = append(meetings, m)
	}
	return meetings, rows.Err()
}

// Update loads the Meeting under a row lock, hands it to fn for mutation,
// and persists the result — all in one transaction, so concurrent Updates
// serialize on the Meeting row.
func (s *Meetings) Update(ctx context.Context, id domain.MeetingID, fn func(*domain.Meeting) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	snapshot, err := s.scanMeeting(ctx, tx, id, true)
	if err != nil {
		return err
	}

	m, err := reconstitute(id, snapshot)
	if err != nil {
		return err
	}
	if err := fn(&m); err != nil {
		return err
	}

	snapshot = m.Snapshot()
	if _, err := tx.Exec(ctx,
		"UPDATE meetings SET title = $2, start = $3, duration_minutes = $4, description = $5, cancelled_at = $6 WHERE id = $1::uuid",
		snapshot.ID, snapshot.Title, snapshot.Start, minutes(snapshot.Duration), snapshot.Description, snapshot.CancelledAt); err != nil {
		return fmt.Errorf("update meeting: %w", err)
	}
	if _, err := tx.Exec(ctx, "DELETE FROM meeting_guests WHERE meeting_id = $1::uuid", id); err != nil {
		return fmt.Errorf("clear guests: %w", err)
	}
	for _, g := range snapshot.Guests {
		if _, err := tx.Exec(ctx,
			"INSERT INTO meeting_guests (meeting_id, user_id) VALUES ($1::uuid, $2::uuid)", id, g); err != nil {
			return fmt.Errorf("add guest: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// querier is satisfied by both *pgxpool.Pool and pgx.Tx.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// scanMeeting loads the Meeting row and its Guests. With forUpdate it takes
// a row lock, so it must run inside a transaction.
func (s *Meetings) scanMeeting(ctx context.Context, q querier, id domain.MeetingID, forUpdate bool) (domain.MeetingSnapshot, error) {
	query := "SELECT " + meetingSelect + " FROM meetings WHERE id = $1::uuid"
	if forUpdate {
		query += " FOR UPDATE"
	}
	snapshot, err := scanMeetingRow(q.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MeetingSnapshot{}, domain.ErrMeetingNotFound
	}
	if err != nil {
		return domain.MeetingSnapshot{}, fmt.Errorf("load meeting: %w", err)
	}
	snapshot.Guests, err = s.guestIDs(ctx, q, id)
	return snapshot, err
}

func scanMeetingRow(row pgx.Row) (domain.MeetingSnapshot, error) {
	var snapshot domain.MeetingSnapshot
	var durationMin int
	err := row.Scan(&snapshot.ID, &snapshot.OwnerID, &snapshot.Title, &snapshot.Start, &durationMin, &snapshot.Description, &snapshot.CancelledAt)
	snapshot.Duration = time.Duration(durationMin) * time.Minute
	return snapshot, err
}

// reconstitute rebuilds a Meeting from its persisted state; a failure means
// the stored data is corrupt.
func reconstitute(id domain.MeetingID, snapshot domain.MeetingSnapshot) (domain.Meeting, error) {
	m, err := domain.MeetingFromSnapshot(snapshot)
	if err != nil {
		return domain.Meeting{}, fmt.Errorf("stored meeting %s is corrupt: %w", id, err)
	}
	return m, nil
}

func (s *Meetings) guestIDs(ctx context.Context, q querier, meetingID domain.MeetingID) ([]domain.UserID, error) {
	rows, err := q.Query(ctx,
		"SELECT user_id::text FROM meeting_guests WHERE meeting_id = $1::uuid ORDER BY user_id", meetingID)
	if err != nil {
		return nil, fmt.Errorf("get guests: %w", err)
	}
	defer rows.Close()

	var guests []domain.UserID
	for rows.Next() {
		var id domain.UserID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		guests = append(guests, id)
	}
	return guests, rows.Err()
}

func minutes(d time.Duration) int {
	return int(d / time.Minute)
}
