// Package migrations embeds the SQL migration files so the binary carries its
// own schema and applies it at startup.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
