package postgres_driver

import (
	"testing"
	"testing/fstest"
)

func TestNonTransactionalSQL(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want bool
	}{
		{"plain ddl", "ALTER TABLE users ADD COLUMN IF NOT EXISTS x int;", false},
		{"plpgsql block", "DO $$ BEGIN\n  PERFORM 1;\nEND $$;", false},
		{"explicit commit", "CREATE TABLE a(); \nCOMMIT;\nCREATE TABLE b();", true},
		{"explicit begin", "BEGIN;\nCREATE TABLE a();", true},
		{"concurrent index", "CREATE INDEX CONCURRENTLY i ON t(c);", true},
		{"vacuum", "VACUUM t;", true},
	}
	for _, tc := range cases {
		if got := nonTransactionalSQL.MatchString(tc.sql); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestPreviousVersion(t *testing.T) {
	embedded := map[uint]bool{1: true, 2: true, 3: true}
	if got := previousVersion(3, embedded); got != 2 {
		t.Errorf("previousVersion(3) = %d, want 2", got)
	}
	if got := previousVersion(1, embedded); got != nilVersion {
		t.Errorf("previousVersion(1) = %d, want NilVersion", got)
	}
}

func TestReadUpSQL(t *testing.T) {
	fsys := fstest.MapFS{
		"000001_a.up.sql":   {Data: []byte("SELECT 1;")},
		"000001_a.down.sql": {Data: []byte("SELECT 0;")},
	}
	got, err := readUpSQL(fsys, 1)
	if err != nil || got != "SELECT 1;" {
		t.Fatalf("readUpSQL(1) = %q, %v", got, err)
	}
	if _, err := readUpSQL(fsys, 2); err == nil {
		t.Fatal("readUpSQL(2) should fail for a missing migration")
	}
}
