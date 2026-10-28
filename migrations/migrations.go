package migrations

import "embed"

// EmbedFS contains the SQL migration files, compiled into the binary.
//
//go:embed *.sql
var EmbedFS embed.FS
