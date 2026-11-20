package postgres_driver_test

import (
	"context"
	"io/fs"
	"os"
	"strings"
	"testing"

	"api-aprop/internal/drivers/postgres_driver"
	"api-aprop/migrations"
	"api-aprop/supabase"
)

// TestMigrateFSWithSupabase_Integration exercises end-to-end migration and
// bidirectional Supabase synchronization against a real PostgreSQL instance.
// Set TEST_POSTGRES_URL to run it (skipped automatically when unset).
func TestMigrateFSWithSupabase_Integration(t *testing.T) {
	connStr := os.Getenv("TEST_POSTGRES_URL")
	if connStr == "" {
		t.Skip("TEST_POSTGRES_URL not set")
	}

	// last is the newest embedded migration: the test follows the repo instead of a hardcoded number.
	entries, err := fs.ReadDir(migrations.EmbedFS, ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}
	last := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			last++
		}
	}
	if last < 2 {
		t.Fatalf("need at least 2 migrations, found %d", last)
	}

	ctx := context.Background()
	drv := postgres_driver.New()
	if err := drv.Connect(connStr, ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer drv.Close()

	// Create a simulated Supabase schema_migrations table.
	if _, err := drv.Pool().Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS supabase_migrations;
		CREATE TABLE IF NOT EXISTS supabase_migrations.schema_migrations (
			version text NOT NULL PRIMARY KEY,
			statements text[],
			name text
		);
	`); err != nil {
		t.Fatalf("create supabase_migrations: %v", err)
	}

	// 1. Fresh run: golang-migrate applies 1..last and reverse-syncs them into Supabase.
	if err := drv.MigrateFSWithSupabase(migrations.EmbedFS, supabase.MigrationsFS); err != nil {
		t.Fatalf("initial MigrateFSWithSupabase: %v", err)
	}

	var supaCount int
	if err := drv.Pool().QueryRow(ctx,
		"SELECT count(*) FROM supabase_migrations.schema_migrations",
	).Scan(&supaCount); err != nil {
		t.Fatalf("count supabase migrations: %v", err)
	}
	if supaCount != last {
		t.Fatalf("supabase_migrations count = %d, want %d", supaCount, last)
	}

	// 2. Simulate Render stuck in a dirty last version while Supabase already has it.
	if _, err := drv.Pool().Exec(ctx,
		"UPDATE public.schema_migrations SET version = $1, dirty = true", last,
	); err != nil {
		t.Fatalf("set dirty %d: %v", last, err)
	}
	if err := drv.MigrateFSWithSupabase(migrations.EmbedFS, supabase.MigrationsFS); err != nil {
		t.Fatalf("healing dirty %d with Supabase present failed: %v", last, err)
	}

	var ver int
	var dirty bool
	if err := drv.Pool().QueryRow(ctx,
		"SELECT version, dirty FROM public.schema_migrations",
	).Scan(&ver, &dirty); err != nil {
		t.Fatalf("read public.schema_migrations: %v", err)
	}
	if ver != last || dirty {
		t.Fatalf("after healing: version=%d dirty=%v, want (%d, false)", ver, dirty, last)
	}

	// 3. Simulate Supabase pushed the last migration ahead of golang-migrate (at last-1).
	if _, err := drv.Pool().Exec(ctx,
		"UPDATE public.schema_migrations SET version = $1, dirty = false", last-1,
	); err != nil {
		t.Fatalf("set version %d: %v", last-1, err)
	}
	if err := drv.MigrateFSWithSupabase(migrations.EmbedFS, supabase.MigrationsFS); err != nil {
		t.Fatalf("catching up from %d to %d failed: %v", last-1, last, err)
	}
	if err := drv.Pool().QueryRow(ctx,
		"SELECT version, dirty FROM public.schema_migrations",
	).Scan(&ver, &dirty); err != nil {
		t.Fatalf("read public.schema_migrations: %v", err)
	}
	if ver != last || dirty {
		t.Fatalf("after catch-up: version=%d dirty=%v, want (%d, false)", ver, dirty, last)
	}

	// 4. Genuine dirty failure NOT present in Supabase must still fail loudly.
	// The last up file is transactional, so recoverDirty forces the previous version and retries it.
	if _, err := drv.Pool().Exec(ctx,
		"DELETE FROM supabase_migrations.schema_migrations WHERE right(version, 6) = lpad($1::int::text, 6, '0')", last,
	); err != nil {
		t.Fatalf("remove %d from supabase: %v", last, err)
	}
	if _, err := drv.Pool().Exec(ctx,
		"UPDATE public.schema_migrations SET version = $1, dirty = true", last,
	); err != nil {
		t.Fatalf("set dirty %d: %v", last, err)
	}
	// Either the retry works, or it fails with the real SQL error: never with the opaque dirty error.
	if err := drv.MigrateFSWithSupabase(migrations.EmbedFS, supabase.MigrationsFS); err != nil &&
		strings.Contains(err.Error(), "Dirty database version") {
		t.Fatalf("a rolled-back dirty migration should be retried, got: %v", err)
	}
}
