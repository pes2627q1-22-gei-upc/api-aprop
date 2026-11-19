package supabase

import "embed"

// MigrationsFS contains the Supabase SQL migration files, used to keep
// supabase_migrations.schema_migrations in sync with golang-migrate.
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
