package preference_handler

import (
	"encoding/json"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/models/preference_model"
	"api-aprop/internal/services/preference_service"
)

const UserIDHeader = "X-User-ID"

type PreferenceHandler struct {
	svc preference_service.Service
}

func NewPreferenceHandler(svc preference_service.Service) *PreferenceHandler {
	return &PreferenceHandler{svc: svc}
}

func (h *PreferenceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /preferences/test", h.GetQuestions)
	mux.HandleFunc("POST /preferences/test", h.SubmitTest)
	mux.HandleFunc("GET /preferences", h.GetUserPreferences)
}

func (h *PreferenceHandler) GetQuestions(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get(UserIDHeader)
	if userID == "" {
		response.WriteError(w, http.StatusBadRequest, "missing "+UserIDHeader+" header")
		return
	}

	questions, err := h.svc.GetQuestions(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, questions)
}

func (h *PreferenceHandler) SubmitTest(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get(UserIDHeader)
	if userID == "" {
		response.WriteError(w, http.StatusBadRequest, "missing "+UserIDHeader+" header")
		return
	}

	var req preference_model.SubmitTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.svc.SubmitTest(r.Context(), userID, req.Answers)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, res)
}

func (h *PreferenceHandler) GetUserPreferences(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get(UserIDHeader)
	if userID == "" {
		response.WriteError(w, http.StatusBadRequest, "missing "+UserIDHeader+" header")
		return
	}

	prefs, err := h.svc.GetUserPreferences(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, prefs)
}
