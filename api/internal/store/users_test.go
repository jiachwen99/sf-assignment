package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

func TestRegisteringAndSigningIn(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made, err := s.CreateUser(ctx, "priya@example.com", "Priya", "a long enough password")
	require.NoError(t, err)
	require.Equal(t, "Priya", made.Name)

	signedIn, err := s.Authenticate(ctx, "priya@example.com", "a long enough password")
	require.NoError(t, err)
	require.Equal(t, made.ID, signedIn.ID)
}

// The password is never stored, and the struct that leaves this package has
// nowhere to put it even by accident.
func TestThePasswordIsHashed(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	_, err := s.CreateUser(ctx, "marcus@example.com", "Marcus", "a long enough password")
	require.NoError(t, err)

	var hash string
	require.NoError(t, s.pool.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE lower(email) = 'marcus@example.com'`).Scan(&hash))
	require.NotContains(t, hash, "a long enough password")
	require.Regexp(t, `^\$2[aby]\$`, hash, "a bcrypt hash, not a digest or the password itself")
}

// Distinguishing the two would turn the login form into a way of finding out
// which addresses are registered.
func TestAnUnknownEmailAndAWrongPasswordGiveTheSameAnswer(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	_, err := s.CreateUser(ctx, "wei@example.com", "Wei", "a long enough password")
	require.NoError(t, err)

	_, wrongPassword := s.Authenticate(ctx, "wei@example.com", "not the password")
	_, noSuchAccount := s.Authenticate(ctx, "nobody@example.com", "not the password")

	require.Error(t, wrongPassword)
	require.Error(t, noSuchAccount)
	require.Equal(t, wrongPassword.Error(), noSuchAccount.Error())
}

// And it must not answer measurably faster either, which is what the dummy
// compare is for. The bound is loose on purpose: this is checking that a bcrypt
// comparison happened at all, not measuring one.
func TestAMissingAccountDoesNotAnswerFaster(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	_, err := s.CreateUser(ctx, "sofia@example.com", "Sofia", "a long enough password")
	require.NoError(t, err)

	start := time.Now()
	_, _ = s.Authenticate(ctx, "sofia@example.com", "not the password")
	wrongPassword := time.Since(start)

	start = time.Now()
	_, _ = s.Authenticate(ctx, "nobody@example.com", "not the password")
	noSuchAccount := time.Since(start)

	require.Greater(t, noSuchAccount*4, wrongPassword,
		"a missing account returned in %s against %s for a wrong password, which is a timing oracle",
		noSuchAccount, wrongPassword)
}

func TestEmailIsUniqueRegardlessOfCase(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	_, err := s.CreateUser(ctx, "daniel@example.com", "Daniel", "a long enough password")
	require.NoError(t, err)

	var invalid *domain.ValidationError
	_, err = s.CreateUser(ctx, "DANIEL@example.com", "Daniel Again", "a long enough password")
	require.ErrorAs(t, err, &invalid)
	require.Contains(t, invalid.Fields["email"], "already registered")
}

func TestSigningInIsCaseInsensitive(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made, err := s.CreateUser(ctx, "Amara@Example.com", "Amara", "a long enough password")
	require.NoError(t, err)

	signedIn, err := s.Authenticate(ctx, "amara@example.com", "a long enough password")
	require.NoError(t, err)
	require.Equal(t, made.ID, signedIn.ID)
}

func TestASessionResolvesToItsUserAndCanBeEnded(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made, err := s.CreateUser(ctx, "tomas@example.com", "Tomas", "a long enough password")
	require.NoError(t, err)

	token, expires, err := s.CreateSession(ctx, made.ID)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.True(t, expires.After(time.Now()))

	found, err := s.UserForSession(ctx, token)
	require.NoError(t, err)
	require.Equal(t, made.ID, found.ID)

	require.NoError(t, s.DeleteSession(ctx, token))
	_, err = s.UserForSession(ctx, token)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// Checked in the query rather than after it, so a caller that forgets to look
// still cannot use one.
func TestAnExpiredSessionIsNotAccepted(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	made, err := s.CreateUser(ctx, "yuki@example.com", "Yuki", "a long enough password")
	require.NoError(t, err)

	token, _, err := s.CreateSession(ctx, made.ID)
	require.NoError(t, err)

	_, err = s.pool.Exec(ctx,
		`UPDATE sessions SET expires_at = now() - interval '1 hour' WHERE token = $1`, token)
	require.NoError(t, err)

	_, err = s.UserForSession(ctx, token)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAnUnknownTokenIsNotFound(t *testing.T) {
	s := NewTestStore(t)

	_, err := s.UserForSession(context.Background(), "not-a-real-token")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// Identity, never separation: signing in changes who the history names, not
// which tasks exist.
func TestTheHistoryNamesWhoMadeTheChange(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	priya, err := s.CreateUser(ctx, "priya@example.com", "Priya", "a long enough password")
	require.NoError(t, err)

	made := newTodo(t, s, "unattributed")
	asPriya := WithActor(ctx, priya.ID)
	_, err = s.UpdateTodo(asPriya, TodoUpdate{
		ID: made.ID, Version: made.Version,
		Name: "attributed", Status: domain.InProgress, Priority: domain.Medium,
	})
	require.NoError(t, err)

	events, err := s.Events(ctx, made.ID)
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Nil(t, events[0].ActorID, "created while nobody was signed in")
	require.NotNil(t, events[1].ActorID)
	require.Equal(t, priya.ID, *events[1].ActorID)
}

// Deleting an account must not delete the record of what it did, which is why
// the reference is ON DELETE SET NULL rather than CASCADE.
func TestDeletingAnAccountLeavesItsHistoryBehind(t *testing.T) {
	s := NewTestStore(t)
	ctx := context.Background()

	kofi, err := s.CreateUser(ctx, "kofi@example.com", "Kofi", "a long enough password")
	require.NoError(t, err)

	made := newTodo(t, s, "outlives its author")
	asKofi := WithActor(ctx, kofi.ID)
	_, err = s.UpdateTodo(asKofi, TodoUpdate{
		ID: made.ID, Version: made.Version,
		Name: "outlives its author", Status: domain.InProgress, Priority: domain.Medium,
	})
	require.NoError(t, err)

	_, err = s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, kofi.ID)
	require.NoError(t, err)

	events, err := s.Events(ctx, made.ID)
	require.NoError(t, err)
	require.Len(t, events, 2, "the change is still recorded")
	require.Nil(t, events[1].ActorID, "it just no longer names anybody")
}
