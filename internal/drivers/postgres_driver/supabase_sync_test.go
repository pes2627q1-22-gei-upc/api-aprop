package postgres_driver

import (
	"reflect"
	"testing"
	"testing/fstest"
)

func TestParseSupabaseVersionSeq(t *testing.T) {
	tests := []struct {
		version string
		wantSeq uint
		wantOK  bool
	}{
		{"20261023000001", 1, true},
		{"20261119000019", 19, true},
		{"20261119000123", 123, true},
		{"20261119000000", 0, false},
		{"20261119", 0, false},
		{"20261119000019_extra", 0, false},
		{"", 0, false},
	}

	for _, tc := range tests {
		got, ok := parseSupabaseVersionSeq(tc.version)
		if ok != tc.wantOK || got != tc.wantSeq {
			t.Errorf("parseSupabaseVersionSeq(%q) = (%d, %v), want (%d, %v)",
				tc.version, got, ok, tc.wantSeq, tc.wantOK)
		}
	}
}

func TestParseEmbeddedUpVersions(t *testing.T) {
	fsys := fstest.MapFS{
		"000001_initial.up.sql":   {Data: []byte("SELECT 1;")},
		"000001_initial.down.sql": {Data: []byte("SELECT 1;")},
		"000019_messages.up.sql":  {Data: []byte("SELECT 1;")},
		"migrations.go":           {Data: []byte("package migrations")},
	}

	got, err := parseEmbeddedUpVersions(fsys)
	if err != nil {
		t.Fatalf("parseEmbeddedUpVersions() error = %v", err)
	}
	want := map[uint]bool{1: true, 19: true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseEmbeddedUpVersions() = %v, want %v", got, want)
	}
}

func TestParseSupabaseMigrationEntries(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/20261119000019_event_messages.sql":       {Data: []byte("SELECT 1;")},
		"migrations/20261113000014_notification_text.up.sql": {Data: []byte("SELECT 1;")},
		"migrations/20261023000001_initial_migration.sql":    {Data: []byte("SELECT 1;")},
		"migrations/README.md":                               {Data: []byte("ignored")},
	}

	got, err := parseSupabaseMigrationEntries(fsys)
	if err != nil {
		t.Fatalf("parseSupabaseMigrationEntries() error = %v", err)
	}

	want := []supabaseMigrationMeta{
		{seq: 1, version: "20261023000001", name: "initial_migration"},
		{seq: 14, version: "20261113000014", name: "notification_text"},
		{seq: 19, version: "20261119000019", name: "event_messages"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseSupabaseMigrationEntries() = %+v, want %+v", got, want)
	}
}

func TestComputeSyncedVersion(t *testing.T) {
	embedded19 := seqSet(1, 19)

	tests := []struct {
		name            string
		current         uint
		dirty           bool
		embedded        map[uint]bool
		supabaseApplied map[uint]bool
		wantTarget      uint
		wantForce       bool
	}{
		{
			name:            "supabase ahead by one migration",
			current:         18,
			dirty:           false,
			embedded:        embedded19,
			supabaseApplied: seqSet(1, 19),
			wantTarget:      19,
			wantForce:       true,
		},
		{
			name:            "heals dirty version already applied by supabase",
			current:         19,
			dirty:           true,
			embedded:        embedded19,
			supabaseApplied: seqSet(1, 19),
			wantTarget:      19,
			wantForce:       true,
		},
		{
			name:            "heals dirty version and catches up further contiguous versions",
			current:         18,
			dirty:           true,
			embedded:        embedded19,
			supabaseApplied: seqSet(1, 19),
			wantTarget:      19,
			wantForce:       true,
		},
		{
			name:            "preserves genuine dirty failure not present in supabase",
			current:         19,
			dirty:           true,
			embedded:        embedded19,
			supabaseApplied: seqSet(1, 18),
			wantTarget:      19,
			wantForce:       false,
		},
		{
			name:            "fresh golang-migrate catches up to full supabase schema",
			current:         0,
			dirty:           false,
			embedded:        embedded19,
			supabaseApplied: seqSet(1, 19),
			wantTarget:      19,
			wantForce:       true,
		},
		{
			name:            "caps synced version at highest embedded migration",
			current:         18,
			dirty:           false,
			embedded:        embedded19,
			supabaseApplied: seqSet(1, 22),
			wantTarget:      19,
			wantForce:       true,
		},
		{
			name:            "stops before gap in supabase migrations",
			current:         0,
			dirty:           false,
			embedded:        embedded19,
			supabaseApplied: map[uint]bool{1: true, 2: true, 4: true},
			wantTarget:      2,
			wantForce:       true,
		},
		{
			name:            "no-op when golang-migrate is already ahead of supabase",
			current:         19,
			dirty:           false,
			embedded:        embedded19,
			supabaseApplied: seqSet(1, 18),
			wantTarget:      19,
			wantForce:       false,
		},
		{
			name:            "no-op when both are equal and clean",
			current:         19,
			dirty:           false,
			embedded:        embedded19,
			supabaseApplied: seqSet(1, 19),
			wantTarget:      19,
			wantForce:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotTarget, gotForce := computeSyncedVersion(tc.current, tc.dirty, tc.embedded, tc.supabaseApplied)
			if gotTarget != tc.wantTarget || gotForce != tc.wantForce {
				t.Errorf("computeSyncedVersion(%d, %v) = (%d, %v), want (%d, %v)",
					tc.current, tc.dirty, gotTarget, gotForce, tc.wantTarget, tc.wantForce)
			}
		})
	}
}

func TestPendingSupabaseRecords(t *testing.T) {
	metas := []supabaseMigrationMeta{
		{seq: 18, version: "20261118000018", name: "unofficial_event_state"},
		{seq: 19, version: "20261119000019", name: "event_messages"},
		{seq: 20, version: "20261120000020", name: "future_migration"},
	}

	got := pendingSupabaseRecords(
		19,
		metas,
		map[uint]bool{18: true},
		map[string]bool{"20261118000018": true},
	)

	want := []supabaseMigrationMeta{
		{seq: 19, version: "20261119000019", name: "event_messages"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("pendingSupabaseRecords() = %+v, want %+v", got, want)
	}
}

func seqSet(from, to uint) map[uint]bool {
	s := make(map[uint]bool, to-from+1)
	for i := from; i <= to; i++ {
		s[i] = true
	}
	return s
}
