package postgres_driver

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

var ErrNotConnected = errors.New("postgres driver is not connected")

// Driver owns the connection pool to Postgres.
type Driver struct {
	pool *pgxpool.Pool
}

func New() *Driver {
	return &Driver{}
}

// Connect opens the connection pool and checks that the database is reachable.
func (d *Driver) Connect(connString string, ctx context.Context) error {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("ping database: %w", err)
	}
	d.pool = pool
	return nil
}

// Pool returns the connection pool, to be used by the repositories.
func (d *Driver) Pool() *pgxpool.Pool {
	return d.pool
}

func (d *Driver) Close() {
	if d.pool != nil {
		d.pool.Close()
	}
}

// MigrateFS applies all pending up migrations found in fsys.
// It reuses the existing pool instead of connString because golang-migrate would send
// pgxpool-only options (like pool_max_conns) to Postgres as runtime parameters.
func (d *Driver) MigrateFS(fsys fs.FS, connString string) error {
	return d.MigrateFSWithSupabase(fsys, nil)
}

// MigrateFSWithSupabase applies all pending up migrations found in fsys while
// keeping public.schema_migrations synchronized with supabase_migrations.schema_migrations
// when connected to a Supabase database.
func (d *Driver) MigrateFSWithSupabase(fsys fs.FS, supabaseFS fs.FS) error {
	if d.pool == nil {
		return ErrNotConnected
	}

	embeddedVersions, err := parseEmbeddedUpVersions(fsys)
	if err != nil {
		return err
	}

	source, err := iofs.New(fsys, ".")
	if err != nil {
		return fmt.Errorf("open migrations: %w", err)
	}

	// Closing this *sql.DB does not close the pool
	db := stdlib.OpenDBFromPool(d.pool)
	database, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		db.Close()
		return fmt.Errorf("create migrate driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "pgx5", database)
	if err != nil {
		db.Close()
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	supaState := syncFromSupabase(d.pool, m, embeddedVersions)
	recoverDirty(m, fsys, embeddedVersions)

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	syncToSupabase(d.pool, m, supaState, supabaseFS)
	return nil
}
