package preference_handler

import (
	"encoding/json"
	"net/http"

	"api-aprop/internal/handlers/language"
	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/handlers/response"
	"api-aprop/internal/models/preference_model"
	"api-aprop/internal/services/preference_service"
	"api-aprop/internal/services/translation_service"
)

type PreferenceHandler struct {
	svc        preference_service.Service
	resolver   requester.Resolver
	translator translation_service.Service // optional: without it questions stay in the original language
}

func NewPreferenceHandler(svc preference_service.Service, resolver requester.Resolver) *PreferenceHandler {
	return &PreferenceHandler{svc: svc, resolver: resolver}
}

// WithTranslator sets the service that translates the test questions of GET /preferences/test.
func (h *PreferenceHandler) WithTranslator(t translation_service.Service) *PreferenceHandler {
	h.translator = t
	return h
}

func (h *PreferenceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /preferences/test", h.GetQuestions)
	mux.HandleFunc("GET /preferences/test/status", h.GetTestStatus)
	mux.HandleFunc("POST /preferences/test/prompted", h.MarkTestPrompted)
	mux.HandleFunc("POST /preferences/test", h.SubmitTest)
	mux.HandleFunc("GET /preferences", h.GetUserPreferences)
}

func (h *PreferenceHandler) GetQuestions(w http.ResponseWriter, r *http.Request) {
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}
	lang, ok := language.Resolve(w, r, h.translator, h.resolver)
	if !ok {
		return
	}

	questions, err := h.svc.GetQuestions(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if h.translator != nil {
		questions = h.translator.TranslateTestQuestions(r.Context(), lang, questions)
	}
	response.WriteJSON(w, http.StatusOK, questions)
}

func (h *PreferenceHandler) GetTestStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	status, err := h.svc.GetTestStatus(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, status)
}

func (h *PreferenceHandler) MarkTestPrompted(w http.ResponseWriter, r *http.Request) {
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	if err := h.svc.MarkTestPrompted(r.Context(), userID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PreferenceHandler) SubmitTest(w http.ResponseWriter, r *http.Request) {
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	var req preference_model.SubmitTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var (
		res preference_model.SubmitTestResponse
		err error
	)
	if req.Partial {
		res, err = h.svc.SavePartialTest(r.Context(), userID, req.Answers)
	} else {
		res, err = h.svc.SubmitTest(r.Context(), userID, req.Answers)
	}
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, res)
}

func (h *PreferenceHandler) GetUserPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	prefs, err := h.svc.GetUserPreferences(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, prefs)
}
