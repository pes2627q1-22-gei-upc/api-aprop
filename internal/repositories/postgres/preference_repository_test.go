package postgres

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"api-aprop/internal/models/user_model"
)

func TestParseCategories(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "empty returns empty non-nil slice",
			raw:  "",
			want: []string{},
		},
		{
			name: "preserves raw gencat labels and trims spaces",
			raw:  " agenda:categories/dansa , agenda:categories/musica , ",
			want: []string{"agenda:categories/dansa", "agenda:categories/musica"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCategories(tt.raw)
			if got == nil {
				t.Fatal("expected non-nil slice")
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("parseCategories(%q) = %#v, want %#v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestPreferenceRepository_GetQuestions_NonPositiveLimit(t *testing.T) {
	r := &PreferenceRepository{}
	got, err := r.GetQuestions(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("got %#v, want empty non-nil slice", got)
	}

	gotExcl, err := r.GetQuestionsExcluding(context.Background(), 0, []int{1, 2})
	if err != nil {
		t.Fatalf("GetQuestionsExcluding unexpected error: %v", err)
	}
	if gotExcl == nil || len(gotExcl) != 0 {
		t.Fatalf("GetQuestionsExcluding got %#v, want empty non-nil slice", gotExcl)
	}
}

func TestPreferenceRepository_InvalidUUID(t *testing.T) {
	r := &PreferenceRepository{}

	if _, err := r.GetTestStatus(context.Background(), "not-a-uuid"); !errors.Is(err, user_model.ErrInvalidID) {
		t.Errorf("GetTestStatus error = %v, want %v", err, user_model.ErrInvalidID)
	}
	if err := r.MarkTestPrompted(context.Background(), "not-a-uuid"); !errors.Is(err, user_model.ErrInvalidID) {
		t.Errorf("MarkTestPrompted error = %v, want %v", err, user_model.ErrInvalidID)
	}
	if _, err := r.GetTestProgress(context.Background(), "not-a-uuid"); !errors.Is(err, user_model.ErrInvalidID) {
		t.Errorf("GetTestProgress error = %v, want %v", err, user_model.ErrInvalidID)
	}
	if err := r.SavePartialProgress(context.Background(), "not-a-uuid", time.Now(), nil); !errors.Is(err, user_model.ErrInvalidID) {
		t.Errorf("SavePartialProgress error = %v, want %v", err, user_model.ErrInvalidID)
	}
	if err := r.SaveTestAndPreferences(context.Background(), "not-a-uuid", time.Now(), nil, nil); !errors.Is(err, user_model.ErrInvalidID) {
		t.Errorf("SaveTestAndPreferences error = %v, want %v", err, user_model.ErrInvalidID)
	}
}
