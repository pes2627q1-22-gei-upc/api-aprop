package postgres

import "testing"

func TestLikeEscaperEscapesWildcards(t *testing.T) {
	tests := map[string]string{
		"joan":      "joan",
		"joan_p":    `joan\_p`,
		"50%":       `50\%`,
		`a\b`:       `a\\b`,
		`%_\`:       `\%\_\\`,
		"Núria Gil": "Núria Gil",
	}
	for in, want := range tests {
		if got := likeEscaper.Replace(in); got != want {
			t.Errorf("escape(%q) = %q, want %q", in, got, want)
		}
	}
}
