package user_model

import (
	"errors"
	"testing"
)

func TestNormalizeID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		want    string
		wantErr bool
	}{
		{name: "canonical", id: "11111111-2222-3333-4444-555555555555", want: "11111111-2222-3333-4444-555555555555"},
		{name: "uppercase is lowercased", id: "AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE", want: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"},
		{name: "empty", id: "", wantErr: true},
		{name: "firebase uid", id: "aBcDeFgHiJkLmNoPqRsTuVwXyZ12", wantErr: true},
		{name: "without dashes", id: "11111111222233334444555555555555", wantErr: true},
		{name: "dash in wrong place", id: "1111111-12222-3333-4444-555555555555", wantErr: true},
		{name: "non hex digit", id: "g1111111-2222-3333-4444-555555555555", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeID(tt.id)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidID) {
					t.Fatalf("NormalizeID(%q) error = %v, want ErrInvalidID", tt.id, err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("NormalizeID(%q) = %q, %v; want %q", tt.id, got, err, tt.want)
			}
		})
	}
}

func TestParseID(t *testing.T) {
	uid, err := ParseID("AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE")
	if err != nil {
		t.Fatalf("ParseID: %v", err)
	}
	if !uid.Valid || uid.String() != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Fatalf("ParseID = %v (valid %v), want the same UUID", uid.String(), uid.Valid)
	}
	if _, err := ParseID("not-a-uuid"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("ParseID(invalid) error = %v, want ErrInvalidID", err)
	}
}
