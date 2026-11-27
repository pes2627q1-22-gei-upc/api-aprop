package assistant_handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetSuggestions(t *testing.T) {
	for query, want := range map[string]string{"?lang=en": "weekend", "?lang=es": "fin de semana", "": "cap de setmana", "?lang=xx": "cap de setmana"} {
		rec := httptest.NewRecorder()
		(&AssistantHandler{}).GetSuggestions(rec, httptest.NewRequest(http.MethodGet, "/assistant/suggestions"+query, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), want) {
			t.Errorf("%q: %d %s", query, rec.Code, rec.Body.String())
		}
	}
}
