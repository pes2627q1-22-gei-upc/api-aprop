package user_handler

import (
	"encoding/json"
	"mime"
	"net/http"
	"path/filepath"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/middleware/auth_middleware"
	"api-aprop/internal/services/user_service"
)

const (
	UserIDHeader  = "X-User-ID"
	maxUploadSize = 5 << 20 // 5 MB
)

type UserHandler struct {
	userService user_service.Service
}

func NewUserHandler(userService user_service.Service) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /users/me", h.GetMyProfile)
	mux.HandleFunc("PATCH /users/me", h.UpdateMyProfile)
	mux.HandleFunc("POST /users/me/avatar", h.UploadAvatar)
	mux.HandleFunc("GET /users/{id}", h.GetUserProfile)
	mux.HandleFunc("GET /users/{id}/created-events", h.ListCreatedEvents)
	mux.HandleFunc("GET /users/{id}/friends", h.GetUserFriends)
	//TODO
	//mux.HandleFunc("POST /users/{id}/friend-requests", h.SendFriendRequest)
	//mux.HandleFunc("GET /users/{id}/friend-requests", h.GetFriendRequests)
	//mux.HandleFunc("POST /friend-requests/{id}/accept", h.AcceptFriendRequest)
	//mux.HandleFunc("POST /friend-requests/{id}/deny", h.DenyFriendRequest)
	//mux.HandleFunc("POST /users/{id}/unfriend", h.Unfriend)
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

func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	requesterID := getRequesterID(r)
	if requesterID == "" {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized: missing auth token or "+UserIDHeader)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		response.WriteError(w, http.StatusBadRequest, "file too large: maximum size is 5MB")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		// Fallback to "file" field name
		file, header, err = r.FormFile("file")
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, "missing 'avatar' or 'file' multipart field")
			return
		}
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = mime.TypeByExtension(filepath.Ext(header.Filename))
	}

	profile, err := h.userService.UploadAvatar(r.Context(), requesterID, header.Filename, contentType, file)
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

func (h *UserHandler) GetUserFriends(w http.ResponseWriter, r *http.Request){
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

	friends, err := h.userService.GetUserFriends(r.Context(), id)
	if err!=nil{
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, friends)
}

func getRequesterID(r *http.Request) string {
	if u, ok := auth_middleware.UserFromContext(r.Context()); ok && u.UID != "" {
		return u.UID
	}
	return r.Header.Get(UserIDHeader)
}