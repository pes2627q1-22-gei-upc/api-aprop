package migrations_test

import (
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"api-aprop/migrations"
	"api-aprop/supabase"
)

var (
	upFileRe       = regexp.MustCompile(`^(\d{6})_(.+)\.up\.sql$`)
	downFileRe     = regexp.MustCompile(`^(\d{6})_(.+)\.down\.sql$`)
	supabaseFileRe = regexp.MustCompile(`^\d{8}(\d{6})_(.+?)(?:\.up)?\.sql$`)
)

// TestMigrationsInSyncWithSupabase guards against drift between migrations/
// (embedded by golang-migrate and read by sqlc) and supabase/migrations/
// (applied by Supabase CLI).
func TestMigrationsInSyncWithSupabase(t *testing.T) {
	migEntries, err := fs.ReadDir(migrations.EmbedFS, ".")
	if err != nil {
		t.Fatalf("read migrations.EmbedFS: %v", err)
	}

	upFiles := make(map[int]string)
	downFiles := make(map[int]string)
	maxSeq := 0

	for _, entry := range migEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if m := upFileRe.FindStringSubmatch(entry.Name()); m != nil {
			seq, _ := strconv.Atoi(m[1])
			if prev, exists := upFiles[seq]; exists {
				t.Fatalf("duplicate migration sequence %06d: %s and %s", seq, prev, entry.Name())
			}
			upFiles[seq] = entry.Name()
			if seq > maxSeq {
				maxSeq = seq
			}
			continue
		}
		if m := downFileRe.FindStringSubmatch(entry.Name()); m != nil {
			seq, _ := strconv.Atoi(m[1])
			if prev, exists := downFiles[seq]; exists {
				t.Fatalf("duplicate down migration sequence %06d: %s and %s", seq, prev, entry.Name())
			}
			downFiles[seq] = entry.Name()
			continue
		}
		t.Errorf("unexpected migration filename in migrations/: %s", entry.Name())
	}

	if maxSeq == 0 {
		t.Fatal("no migrations found in migrations/")
	}

	for seq := 1; seq <= maxSeq; seq++ {
		if _, ok := upFiles[seq]; !ok {
			t.Errorf("missing up migration for sequence %06d in migrations/", seq)
		}
		if _, ok := downFiles[seq]; !ok {
			t.Errorf("missing down migration for sequence %06d in migrations/", seq)
		}
	}

	supaEntries, err := fs.ReadDir(supabase.MigrationsFS, "migrations")
	if err != nil {
		t.Fatalf("read supabase.MigrationsFS: %v", err)
	}

	supaFiles := make(map[int]string)
	for _, entry := range supaEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		m := supabaseFileRe.FindStringSubmatch(entry.Name())
		if m == nil {
			t.Errorf("supabase migration filename %q must follow YYYYMMDD0000XX_<name>.sql format", entry.Name())
			continue
		}
		seq, _ := strconv.Atoi(m[1])
		if prev, exists := supaFiles[seq]; exists {
			t.Fatalf("duplicate supabase migration sequence %06d: %s and %s", seq, prev, entry.Name())
		}
		supaFiles[seq] = entry.Name()
	}

	for seq, upName := range upFiles {
		supaName, ok := supaFiles[seq]
		if !ok {
			t.Errorf("migration %s has no matching file in supabase/migrations/ (expected YYYYMMDD%06d_*.sql)", upName, seq)
			continue
		}

		upBytes, err := fs.ReadFile(migrations.EmbedFS, upName)
		if err != nil {
			t.Fatalf("read migrations/%s: %v", upName, err)
		}
		supaBytes, err := fs.ReadFile(supabase.MigrationsFS, "migrations/"+supaName)
		if err != nil {
			t.Fatalf("read supabase/migrations/%s: %v", supaName, err)
		}

		if normalizeSQL(string(upBytes)) != normalizeSQL(string(supaBytes)) {
			t.Errorf("SQL mismatch between migrations/%s and supabase/migrations/%s", upName, supaName)
		}
	}

	for seq, supaName := range supaFiles {
		if _, ok := upFiles[seq]; !ok {
			t.Errorf("supabase/migrations/%s has no matching migrations/%06d_*.up.sql", supaName, seq)
		}
	}
}

func normalizeSQL(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimSpace(s)
}

// firstIdempotentMigration is the first migration that must be safe to re-run:
// older ones were written before this rule and are already applied everywhere.
const firstIdempotentMigration = 20

// notIdempotentSQL matches DDL that fails when run twice. A migration can be
// applied but not recorded (crash, manual apply), so re-running must be harmless.
var notIdempotentSQL = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bcreate\s+table\s+(?:(?:unlogged|temporary|temp)\s+)?(?:public\.)?"?\w+`),
	regexp.MustCompile(`(?i)\bcreate\s+(?:unique\s+)?index\s+(?:concurrently\s+)?"?\w+`),
	regexp.MustCompile(`(?i)\badd\s+column\s+"?\w+`),
	regexp.MustCompile(`(?i)\bdrop\s+(?:table|column|index|type|constraint)\s+"?\w+`),
}

// TestNewMigrationsAreIdempotent rejects DDL without IF [NOT] EXISTS in
// migrations from firstIdempotentMigration on.
func TestNewMigrationsAreIdempotent(t *testing.T) {
	entries, err := fs.ReadDir(migrations.EmbedFS, ".")
	if err != nil {
		t.Fatalf("read migrations.EmbedFS: %v", err)
	}
	guard := regexp.MustCompile(`(?i)\bif\s+(?:not\s+)?exists\b`)
	for _, entry := range entries {
		m := upFileRe.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		if seq, _ := strconv.Atoi(m[1]); seq < firstIdempotentMigration {
			continue
		}
		b, err := fs.ReadFile(migrations.EmbedFS, entry.Name())
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			for _, re := range notIdempotentSQL {
				if re.MatchString(line) && !guard.MatchString(line) {
					t.Errorf("%s:%d is not idempotent (use IF [NOT] EXISTS): %s", entry.Name(), i+1, strings.TrimSpace(line))
				}
			}
		}
	}
}
