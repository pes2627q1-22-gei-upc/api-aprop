package assistant_handler

import (
	"net/http"

	"api-aprop/internal/handlers/response"
)

// suggestions are example questions the app can show as quick replies, per language.
var suggestions = map[string][]string{
	"ca": {"Què puc fer aquest cap de setmana?", "Concerts gratuïts a Barcelona", "Com m'apunto a una activitat?", "On veig les meves activitats?"},
	"es": {"¿Qué puedo hacer este fin de semana?", "Conciertos gratuitos en Barcelona", "¿Cómo me apunto a una actividad?", "¿Dónde veo mis actividades?"},
	"en": {"What can I do this weekend?", "Free concerts in Barcelona", "How do I sign up for an activity?", "Where can I see my activities?"},
}

// GetSuggestions handles GET /assistant/suggestions?lang=ca|es|en (Catalan by default).
func (h *AssistantHandler) GetSuggestions(w http.ResponseWriter, r *http.Request) {
	list, ok := suggestions[r.URL.Query().Get("lang")]
	if !ok {
		list = suggestions["ca"]
	}
	response.WriteJSON(w, http.StatusOK, map[string][]string{"suggestions": list})
}
