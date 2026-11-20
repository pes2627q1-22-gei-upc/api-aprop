package postgres_driver

import (
	"fmt"
	"io/fs"
	"log/slog"
	"regexp"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
)

// nilVersion is golang-migrate's "no migration applied" version (database.NilVersion).
const nilVersion = -1

// nonTransactionalSQL matches statements that cannot be rolled back with the
// rest of the file: explicit transaction control, CONCURRENTLY and VACUUM.
var nonTransactionalSQL = regexp.MustCompile(`(?im)^\s*(commit|begin|start\s+transaction)\s*;|\bconcurrently\b|\bvacuum\b`)

// readUpSQL returns the content of the up migration for the given version.
func readUpSQL(fsys fs.FS, version uint) (string, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		m := embeddedUpFilePattern.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		if v, err := strconv.ParseUint(m[1], 10, 64); err == nil && uint(v) == version {
			b, err := fs.ReadFile(fsys, entry.Name())
			return string(b), err
		}
	}
	return "", fmt.Errorf("up migration %d not found", version)
}

// previousVersion returns the highest embedded version below current, or
// nilVersion when current is the first one.
func previousVersion(current uint, embedded map[uint]bool) int {
	prev := nilVersion
	for v := range embedded {
		if v < current && int(v) > prev {
			prev = int(v)
		}
	}
	return prev
}

// recoverDirty clears a dirty flag left by a migration that failed halfway.
// golang-migrate runs each file as one multi-statement query, so Postgres
// rolled the whole file back and it is safe to retry it. When the file has
// statements that escape the transaction (see nonTransactionalSQL) the state is
// unknown and we leave it dirty for a human to fix.
func recoverDirty(m *migrate.Migrate, fsys fs.FS, embedded map[uint]bool) {
	current, dirty, err := m.Version()
	if err != nil || !dirty {
		return
	}

	sql, err := readUpSQL(fsys, current)
	if err != nil {
		slog.Warn("Dirty migration state not recovered", "version", current, "error", err)
		return
	}
	if nonTransactionalSQL.MatchString(sql) {
		slog.Error("Dirty migration contains non-transactional statements: fix the database and force the version by hand",
			"version", current)
		return
	}

	prev := previousVersion(current, embedded)
	slog.Warn("Recovering dirty migration: it was rolled back, retrying it", "dirty_version", current, "forcing", prev)
	if err := m.Force(prev); err != nil {
		slog.Warn("Failed to force previous migration version", "version", prev, "error", err)
	}
}
