package user_handler

import (
	"encoding/json"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/services/user_service"
)

const UserIDHeader = "X-User-ID"

type UserHandler struct {
	userService user_service.Service
}

func NewUserHandler(userService user_service.Service) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /users/me", h.GetMyProfile)
	mux.HandleFunc("PATCH /users/me", h.UpdateMyProfile)
	mux.HandleFunc("GET /users/{id}", h.GetUserProfile)
	mux.HandleFunc("GET /users/{id}/created-events", h.ListCreatedEvents)
}

func (h *UserHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	requesterID := getRequesterID(r)
	if requesterID == "" {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized: missing auth token or "+UserIDHeader)
		return
	}

	profile, err := h.userService.GetUserProfile(r.Context(), "me", requesterID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, profile)
}

func (h *UserHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.WriteError(w, http.StatusBadRequest, "missing user id")
		return
	}
	if id == "me" {
		h.GetMyProfile(w, r)
		return
	}

	requesterID := getRequesterID(r)
	profile, err := h.userService.GetUserProfile(r.Context(), id, requesterID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, profile)
}

func (h *UserHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	requesterID := getRequesterID(r)
	if requesterID == "" {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized: missing auth token or "+UserIDHeader)
		return
	}

	var body struct {
		Name           *string `json:"name"`
		Username       *string `json:"username"`
		Biography      *string `json:"biography"`
		ProfilePicture *string `json:"profile-picture"`
		ShowPlans      *bool   `json:"show-plans"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid json payload: "+err.Error())
		return
	}

	profile, err := h.userService.UpdateProfile(r.Context(), requesterID, user_service.UpdateProfileInput{
		Name:           body.Name,
		Username:       body.Username,
		Biography:      body.Biography,
		ProfilePicture: body.ProfilePicture,
		ShowPlans:      body.ShowPlans,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, profile)
}

func (h *UserHandler) ListCreatedEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.WriteError(w, http.StatusBadRequest, "missing user id")
		return
	}
	if id == "me" {
		requesterID := getRequesterID(r)
		if requesterID == "" {
			response.WriteError(w, http.StatusUnauthorized, "unauthorized: missing auth token or "+UserIDHeader)
			return
		}
		id = requesterID
	}

	events, err := h.userService.ListCreatedEvents(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, events)
}

func getRequesterID(r *http.Request) string {
	if u, ok := auth_middleware.UserFromContext(r.Context()); ok && u.UID != "" {
		return u.UID
	}
	return r.Header.Get(UserIDHeader)
}
