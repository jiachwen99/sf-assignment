package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// One container for the whole package, one clean schema per test. A container
// per test would make the suite unusable.
var shared struct {
	dsn string
}

// Boots PostgreSQL once. Nothing needs to be running beforehand.
func RunWithPostgres(m *testing.M) int {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("todo_test"),
		postgres.WithUsername("todo"),
		postgres.WithPassword("todo"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(90*time.Second),
		),
	)
	if err != nil {
		panic("start postgres: " + err.Error())
	}
	defer func() { _ = testcontainers.TerminateContainer(container) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("connection string: " + err.Error())
	}
	shared.dsn = dsn

	return m.Run()
}

func RunTestMain(m *testing.M) {
	os.Exit(RunWithPostgres(m))
}

func NewTestStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()

	s, err := New(ctx, shared.dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(s.Close)

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Both roots, because CASCADE follows references rather than reaching
	// sideways: truncating todos alone leaves accounts behind, and the next
	// test to register the same address is refused as a duplicate.
	if _, err := s.pool.Exec(ctx, `TRUNCATE todos, users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return s
}
