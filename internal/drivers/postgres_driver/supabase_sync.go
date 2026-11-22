package postgres_driver

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	embeddedUpFilePattern    = regexp.MustCompile(`^(\d+)_.+\.up\.sql$`)
	supabaseFilePattern      = regexp.MustCompile(`^(\d{8}(\d{6}))_(.+?)(?:\.up)?\.sql$`)
	supabaseVersionSeqRegexp = regexp.MustCompile(`^\d{8}(\d{6})$`)
)

type supabaseMigrationMeta struct {
	seq     uint
	version string
	name    string
}

type supabaseState struct {
	available   bool
	appliedSeqs map[uint]bool
	appliedRaw  map[string]bool
}

func parseEmbeddedUpVersions(fsys fs.FS) (map[uint]bool, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	versions := make(map[uint]bool, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		m := embeddedUpFilePattern.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		v, err := strconv.ParseUint(m[1], 10, 64)
		if err != nil || v == 0 {
			continue
		}
		versions[uint(v)] = true
	}
	return versions, nil
}

func parseSupabaseMigrationEntries(fsys fs.FS) ([]supabaseMigrationMeta, error) {
	if fsys == nil {
		return nil, nil
	}
	dir := "migrations"
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		entries, err = fs.ReadDir(fsys, ".")
		if err != nil {
			return nil, fmt.Errorf("read supabase migrations: %w", err)
		}
	}

	var metas []supabaseMigrationMeta
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		m := supabaseFilePattern.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		seq, err := strconv.ParseUint(m[2], 10, 64)
		if err != nil || seq == 0 {
			continue
		}
		metas = append(metas, supabaseMigrationMeta{
			seq:     uint(seq),
			version: m[1],
			name:    m[3],
		})
	}
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].seq < metas[j].seq
	})
	return metas, nil
}

func parseSupabaseVersionSeq(version string) (uint, bool) {
	m := supabaseVersionSeqRegexp.FindStringSubmatch(version)
	if m == nil {
		return 0, false
	}
	seq, err := strconv.ParseUint(m[1], 10, 64)
	if err != nil || seq == 0 {
		return 0, false
	}
	return uint(seq), true
}

// computeSyncedVersion decides whether golang-migrate's version should be
// forced to catch up with migrations already recorded by Supabase.
//
// It advances contiguously across versions present in both embedded and
// supabaseApplied. If golang-migrate is dirty, it clears the dirty flag only
// when the dirty version itself is already recorded as applied by Supabase
// (meaning golang-migrate failed trying to re-apply a Supabase migration).
func computeSyncedVersion(
	current uint,
	dirty bool,
	embedded map[uint]bool,
	supabaseApplied map[uint]bool,
) (target uint, shouldForce bool) {
	if dirty {
		if !supabaseApplied[current] {
			return current, false
		}
		target = current
		for embedded[target+1] && supabaseApplied[target+1] {
			target++
		}
		return target, true
	}

	target = current
	for embedded[target+1] && supabaseApplied[target+1] {
		target++
	}
	return target, target > current
}

// pendingSupabaseRecords returns the Supabase migration entries whose sequence
// number is <= finalVer and that are not yet recorded in supabase_migrations.
func pendingSupabaseRecords(
	finalVer uint,
	metas []supabaseMigrationMeta,
	appliedSeqs map[uint]bool,
	appliedRaw map[string]bool,
) []supabaseMigrationMeta {
	var pending []supabaseMigrationMeta
	for _, meta := range metas {
		if meta.seq > finalVer {
			continue
		}
		if appliedSeqs[meta.seq] || appliedRaw[meta.version] {
			continue
		}
		pending = append(pending, meta)
	}
	return pending
}

func readSupabaseState(ctx context.Context, pool *pgxpool.Pool) (supabaseState, error) {
	var exists bool
	if err := pool.QueryRow(ctx,
		"SELECT to_regclass('supabase_migrations.schema_migrations') IS NOT NULL",
	).Scan(&exists); err != nil {
		return supabaseState{}, err
	}
	if !exists {
		return supabaseState{available: false}, nil
	}

	rows, err := pool.Query(ctx, "SELECT version FROM supabase_migrations.schema_migrations")
	if err != nil {
		return supabaseState{}, err
	}
	defer rows.Close()

	state := supabaseState{
		available:   true,
		appliedSeqs: make(map[uint]bool),
		appliedRaw:  make(map[string]bool),
	}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return supabaseState{}, err
		}
		state.appliedRaw[version] = true
		if seq, ok := parseSupabaseVersionSeq(version); ok {
			state.appliedSeqs[seq] = true
		}
	}
	if err := rows.Err(); err != nil {
		return supabaseState{}, err
	}
	return state, nil
}

// syncFromSupabase aligns public.schema_migrations with migrations already
// applied via Supabase CLI before golang-migrate runs m.Up().
func syncFromSupabase(
	pool *pgxpool.Pool,
	m *migrate.Migrate,
	embedded map[uint]bool,
) supabaseState {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	state, err := readSupabaseState(ctx, pool)
	if err != nil {
		slog.Warn("Skipping Supabase migration pre-sync", "error", err)
		return supabaseState{}
	}
	if !state.available {
		return state
	}

	current, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		current = 0
		dirty = false
	} else if err != nil {
		slog.Warn("Skipping Supabase migration pre-sync: cannot read current version", "error", err)
		return state
	}

	target, shouldForce := computeSyncedVersion(current, dirty, embedded, state.appliedSeqs)
	if !shouldForce {
		return state
	}

	slog.Info("Syncing golang-migrate version with Supabase migrations",
		"from", current,
		"was_dirty", dirty,
		"to", target,
	)
	if err := m.Force(int(target)); err != nil {
		slog.Warn("Failed to force synced migration version", "target", target, "error", err)
	}
	return state
}

// syncToSupabase records migrations applied by golang-migrate into
// supabase_migrations.schema_migrations so a later `supabase db push` does not
// attempt to re-apply them. Best-effort: warnings are logged without failing startup.
func syncToSupabase(
	pool *pgxpool.Pool,
	m *migrate.Migrate,
	state supabaseState,
	supabaseFS fs.FS,
) {
	if !state.available || supabaseFS == nil {
		return
	}

	finalVer, dirty, err := m.Version()
	if err != nil || dirty || finalVer == 0 {
		return
	}

	metas, err := parseSupabaseMigrationEntries(supabaseFS)
	if err != nil {
		slog.Warn("Skipping Supabase migration post-sync: cannot parse supabase migrations", "error", err)
		return
	}

	pending := pendingSupabaseRecords(finalVer, metas, state.appliedSeqs, state.appliedRaw)
	if len(pending) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var hasNameCol bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'supabase_migrations'
			  AND table_name = 'schema_migrations'
			  AND column_name = 'name'
		)
	`).Scan(&hasNameCol); err != nil {
		slog.Warn("Skipping Supabase migration post-sync: cannot inspect columns", "error", err)
		return
	}

	for _, meta := range pending {
		var execErr error
		if hasNameCol {
			_, execErr = pool.Exec(ctx,
				"INSERT INTO supabase_migrations.schema_migrations (version, name) VALUES ($1, $2) ON CONFLICT (version) DO NOTHING",
				meta.version, meta.name,
			)
		} else {
			_, execErr = pool.Exec(ctx,
				"INSERT INTO supabase_migrations.schema_migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING",
				meta.version,
			)
		}
		if execErr != nil {
			slog.Warn("Failed to record migration in Supabase schema_migrations",
				"version", meta.version,
				"name", meta.name,
				"error", execErr,
			)
			return
		}
		slog.Info("Recorded migration in Supabase schema_migrations",
			"version", meta.version,
			"name", meta.name,
		)
	}
}
