package store

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

// No password hash on the way out. The field does not exist on the struct
// rather than being tagged to hide it, so it cannot be added to a response by
// somebody reaching for a convenient type.
type User struct {
	ID        int64     `db:"id" json:"id"`
	Email     string    `db:"email" json:"email"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"-"`
}

const insertUser = `
INSERT INTO users (email, name, password_hash)
VALUES ($1, $2, $3)
RETURNING id, email, name, created_at`

func (s *Store) CreateUser(ctx context.Context, email, name, password string) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, wrap("create user", err)
	}

	rows, err := s.pool.Query(ctx, insertUser,
		strings.TrimSpace(email), strings.TrimSpace(name), string(hash))
	if err != nil {
		return User{}, asDuplicateEmail(err)
	}
	// With pgx the constraint violation surfaces here rather than on Query.
	u, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[User])
	if err != nil {
		return User{}, asDuplicateEmail(err)
	}
	return u, nil
}

func asDuplicateEmail(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.Invalid("email", "That email address is already registered")
	}
	return wrap("create user", err)
}

const selectUserForLogin = `
SELECT id, email, name, password_hash, created_at FROM users WHERE lower(email) = lower($1)`

// A bcrypt hash of nothing anybody knows, compared against when no account
// matches so that a missing address does not answer measurably faster than a
// wrong password.
const dummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

/*
 * The same error for an unknown email and a wrong password.
 *
 * Distinguishing them turns the login form into a way of finding out which
 * addresses are registered, which is worth more to somebody attacking this than
 * the marginal helpfulness of a better message is to anybody using it.
 */
func (s *Store) Authenticate(ctx context.Context, email, password string) (User, error) {
	var u User
	var hash string

	err := s.pool.QueryRow(ctx, selectUserForLogin, email).
		Scan(&u.ID, &u.Email, &u.Name, &hash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
		return User{}, errWrongCredentials
	}
	if err != nil {
		return User{}, wrap("authenticate", err)
	}

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return User{}, errWrongCredentials
	}
	return u, nil
}

var errWrongCredentials = domain.Invalid("credentials", "That email and password do not match")

const sessionLifetime = 7 * 24 * time.Hour

const insertSession = `INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`

// The token is 256 bits from crypto/rand, so it is the cookie rather than
// anything derived from the account, and it carries no information.
func (s *Store) CreateSession(ctx context.Context, userID int64) (string, time.Time, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expires := time.Now().Add(sessionLifetime)

	if _, err := s.pool.Exec(ctx, insertSession, token, userID, expires); err != nil {
		return "", time.Time{}, wrap("create session", err)
	}
	return token, expires, nil
}

// Expiry is checked in the query rather than after it, so an expired token
// cannot be used by any caller that forgets to look.
const selectSessionUser = `
SELECT u.id, u.email, u.name, u.created_at
FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.token = $1 AND s.expires_at > now()`

func (s *Store) UserForSession(ctx context.Context, token string) (User, error) {
	rows, err := s.pool.Query(ctx, selectSessionUser, token)
	if err != nil {
		return User{}, wrap("read session", err)
	}
	u, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[User])
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, domain.ErrNotFound
	}
	return u, wrap("read session", err)
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return wrap("delete session", err)
}

const selectUsersByID = `SELECT id, email, name, created_at FROM users WHERE id = ANY($1)`

// Fetched in one query for a whole history rather than one per entry.
func (s *Store) UsersByID(ctx context.Context, ids []int64) ([]User, error) {
	rows, err := s.pool.Query(ctx, selectUsersByID, ids)
	if err != nil {
		return nil, wrap("read users", err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByName[User])
	return out, wrap("read users", err)
}
