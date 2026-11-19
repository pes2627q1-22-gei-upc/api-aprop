package postgres_driver_test

import (
	"context"
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

	// 1. Fresh run: golang-migrate applies 1..19 and reverse-syncs them into Supabase.
	if err := drv.MigrateFSWithSupabase(migrations.EmbedFS, supabase.MigrationsFS); err != nil {
		t.Fatalf("initial MigrateFSWithSupabase: %v", err)
	}

	var supaCount int
	if err := drv.Pool().QueryRow(ctx,
		"SELECT count(*) FROM supabase_migrations.schema_migrations",
	).Scan(&supaCount); err != nil {
		t.Fatalf("count supabase migrations: %v", err)
	}
	if supaCount != 19 {
		t.Fatalf("supabase_migrations count = %d, want 19", supaCount)
	}

	// 2. Simulate Render stuck in Dirty database version 19 while Supabase already has 19.
	if _, err := drv.Pool().Exec(ctx,
		"UPDATE public.schema_migrations SET version = 19, dirty = true",
	); err != nil {
		t.Fatalf("set dirty 19: %v", err)
	}
	if err := drv.MigrateFSWithSupabase(migrations.EmbedFS, supabase.MigrationsFS); err != nil {
		t.Fatalf("healing dirty 19 with Supabase present failed: %v", err)
	}

	var ver int
	var dirty bool
	if err := drv.Pool().QueryRow(ctx,
		"SELECT version, dirty FROM public.schema_migrations",
	).Scan(&ver, &dirty); err != nil {
		t.Fatalf("read public.schema_migrations: %v", err)
	}
	if ver != 19 || dirty {
		t.Fatalf("after healing: version=%d dirty=%v, want (19, false)", ver, dirty)
	}

	// 3. Simulate Supabase pushed migration 19 ahead of golang-migrate (at 18).
	if _, err := drv.Pool().Exec(ctx,
		"UPDATE public.schema_migrations SET version = 18, dirty = false",
	); err != nil {
		t.Fatalf("set version 18: %v", err)
	}
	if err := drv.MigrateFSWithSupabase(migrations.EmbedFS, supabase.MigrationsFS); err != nil {
		t.Fatalf("catching up from 18 to 19 failed: %v", err)
	}
	if err := drv.Pool().QueryRow(ctx,
		"SELECT version, dirty FROM public.schema_migrations",
	).Scan(&ver, &dirty); err != nil {
		t.Fatalf("read public.schema_migrations: %v", err)
	}
	if ver != 19 || dirty {
		t.Fatalf("after catch-up: version=%d dirty=%v, want (19, false)", ver, dirty)
	}

	// 4. Genuine dirty failure NOT present in Supabase must still fail loudly.
	if _, err := drv.Pool().Exec(ctx, `
		DELETE FROM supabase_migrations.schema_migrations WHERE version = '20261119000019';
		UPDATE public.schema_migrations SET version = 19, dirty = true;
	`); err != nil {
		t.Fatalf("setup genuine dirty failure: %v", err)
	}
	err := drv.MigrateFSWithSupabase(migrations.EmbedFS, supabase.MigrationsFS)
	if err == nil || !strings.Contains(err.Error(), "Dirty database version 19") {
		t.Fatalf("expected Dirty database version 19 error, got: %v", err)
	}
}
