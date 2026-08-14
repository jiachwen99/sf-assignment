package service_test

import (
	"os"
	"testing"

	"github.com/jiachwen99/sf-assignment/api/internal/store"
)

// The same container the store tests use. Bulk behaviour is about what happens
// across several real writes, so there is nothing here worth faking.
func TestMain(m *testing.M) {
	os.Exit(store.RunWithPostgres(m))
}
