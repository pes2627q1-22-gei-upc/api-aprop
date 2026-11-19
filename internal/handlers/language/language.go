// Package language resolves the language in which a response is written.
package language

import (
	"log/slog"
	"net/http"

	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/handlers/response"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/models/language_model"
	"api-aprop/internal/services/translation_service"
)

// Resolve returns the language of the response: ?lang= wins; otherwise the language in the settings of the signed in
// user; otherwise the original one (Catalan). An invalid ?lang= writes a 400 response and returns false.
// A nil translator (translation not wired) always returns the original language. A user that cannot be identified
// or whose settings cannot be read only means the original language: the lookup never fails the request.
// TODO(DP-043): confirm the resolution order (?lang, user settings, original).
func Resolve(w http.ResponseWriter, r *http.Request, translator translation_service.Service, resolver requester.Resolver) (string, bool) {
	if translator == nil {
		return language_model.Default, true
	}
	if lang := r.URL.Query().Get("lang"); lang != "" {
		if !language_model.Valid(lang) {
			response.WriteError(w, http.StatusBadRequest, "invalid lang, expected ca, es or en")
			return "", false
		}
		return lang, true
	}

	userID := quietUserID(r, resolver)
	if userID == "" {
		return language_model.Default, true
	}
	lang, err := translator.UserLanguage(r.Context(), userID)
	if err != nil || !language_model.Valid(lang) {
		if err != nil {
			slog.Warn("could not read the language of the user; using the original one", "error", err)
		}
		return language_model.Default, true
	}
	return lang, true
}

// quietUserID is requester.ID without writing error responses: it returns "" when the requester is unknown.
func quietUserID(r *http.Request, resolver requester.Resolver) string {
	if u, ok := auth_middleware.UserFromContext(r.Context()); ok && u.UID != "" {
		id, err := resolver.ResolveUserID(r.Context(), u.UID)
		if err != nil {
			return ""
		}
		return id
	}
	return r.Header.Get(requester.UserIDHeader)
}
