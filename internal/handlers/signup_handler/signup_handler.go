package signup_handler

import (
	"net/http"
	"strconv"
	"time"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/models/signup_model"
	"api-aprop/internal/services/signup_service"
)

// UserIDHeader carries the user identity until authentication exists.
const UserIDHeader = "X-User-ID"

const dateLayout = "2006-01-02"

type SignUpHandler struct {
	signUpService signup_service.Service
}

func NewSignUpHandler(signUpService signup_service.Service) *SignUpHandler {
	return &SignUpHandler{signUpService: signUpService}
}

// RegisterRoutes registers the sign up endpoints.
func (h *SignUpHandler) RegisterRoutes(mux *http.ServeMux) {
	// Sign up handler: users signed up to events (calendar)
	mux.HandleFunc("POST /events/{id}/signup", h.SignUp)
	mux.HandleFunc("GET /users/{userId}/events", h.ListUserEvents)
	mux.HandleFunc("DELETE /events/{id}/enroll", h.Unenroll)
	mux.HandleFunc("DELETE /events/{id}/signup", h.Unenroll)
}

// SignUp handles POST /events/{id}/signup
func (h *SignUpHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}
	// TODO(DP-011): replace X-User-ID with the authenticated Firebase identity.
	userID := r.Header.Get(UserIDHeader)
	if userID == "" {
		response.WriteError(w, http.StatusBadRequest, "missing "+UserIDHeader+" header")
		return
	}

	signUp, err := h.signUpService.SignUp(r.Context(), userID, eventID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	status := http.StatusOK
	if signUp.Created {
		status = http.StatusCreated
	}
	response.WriteJSON(w, status, signUp)
}

// Unenroll handles DELETE /events/{id}/enroll and DELETE /events/{id}/signup
func (h *SignUpHandler) Unenroll(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}
	userID := r.Header.Get(UserIDHeader)
	if userID == "" {
		response.WriteError(w, http.StatusBadRequest, "missing "+UserIDHeader+" header")
		return
	}

	unenroll, err := h.signUpService.Unenroll(r.Context(), userID, eventID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, unenroll)
}

// ListUserEvents handles GET /users/{userId}/events?from=YYYY-MM-DD&to=YYYY-MM-DD
// TODO(DP-011): there is no authorization check that the caller is the requested user.
func (h *SignUpHandler) ListUserEvents(w http.ResponseWriter, r *http.Request) {
	from, ok := parseDateParam(w, r, "from")
	if !ok {
		return
	}
	to, ok := parseDateParam(w, r, "to")
	if !ok {
		return
	}

	events, err := h.signUpService.ListUserEvents(r.Context(), r.PathValue("userId"), signup_model.DateRange{From: from, To: to})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, events)
}

// parseDateParam returns nil when the query parameter is absent and writes a 400 response when it is malformed.
func parseDateParam(w http.ResponseWriter, r *http.Request, name string) (*time.Time, bool) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return nil, true
	}
	date, err := time.Parse(dateLayout, value)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid "+name+" date, expected YYYY-MM-DD")
		return nil, false
	}
	return &date, true
}
