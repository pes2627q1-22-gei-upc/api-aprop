package user_handler

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"time"

	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/user_service"
)

const maxUploadSize = 5 << 20 // 5 MB

type UserHandler struct {
	userService user_service.Service
	resolver    requester.Resolver
}

func NewUserHandler(userService user_service.Service, resolver requester.Resolver) *UserHandler {
	return &UserHandler{userService: userService, resolver: resolver}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /users/me", h.GetMyProfile)
	mux.HandleFunc("PATCH /users/me", h.UpdateMyProfile)
	mux.HandleFunc("DELETE /users/me", h.DeleteMyAccount)
	mux.HandleFunc("POST /users/me/avatar", h.UploadAvatar)
	mux.HandleFunc("GET /users/{id}", h.GetUserProfile)
	mux.HandleFunc("GET /users/{id}/created-events", h.ListCreatedEvents)
	mux.HandleFunc("GET /users/{id}/ratings", h.ListReceivedRatings)
	mux.HandleFunc("GET /users/{id}/friends", h.GetUserFriends)
	mux.HandleFunc("GET /users/me/friend-requests", h.GetFriendRequests)
	mux.HandleFunc("POST /users/{id}/friend-requests", h.SendFriendRequest)
	mux.HandleFunc("POST /users/{id}/friend-requests/{date}/accept", h.AcceptFriendRequest)
	mux.HandleFunc("POST /users/{id}/friend-requests/{date}/deny", h.DenyFriendRequest)
	mux.HandleFunc("POST /users/{id}/unfriend", h.Unfriend)
	mux.HandleFunc("GET /users/me/settings", h.GetSettings)
}

func (h *UserHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	settings, err := h.userService.GetSettings(r.Context(), requesterID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, settings)
}

func (h *UserHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
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

	// Anyone can see a profile; the requester is optional and only used to know the friendship
	var requesterID string
	if requester.Present(r) {
		var ok bool
		if requesterID, ok = requester.ID(w, r, h.resolver); !ok {
			return
		}
	}
	profile, err := h.userService.GetUserProfile(r.Context(), id, requesterID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, profile)
}

func (h *UserHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	var body user_service.UpdateProfileInput

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid json payload: "+err.Error())
		return
	}

	profile, err := h.userService.UpdateProfile(r.Context(), requesterID, body)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, profile)
}

// DeleteMyAccount erases the account of the requester and all its data (RGPD). Responds 204 on success.
func (h *UserHandler) DeleteMyAccount(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	if err := h.userService.DeleteAccount(r.Context(), requesterID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
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
		requesterID, ok := requester.ID(w, r, h.resolver)
		if !ok {
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

func (h *UserHandler) ListReceivedRatings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "me" {
		requesterID, ok := requester.ID(w, r, h.resolver)
		if !ok {
			return
		}
		id = requesterID
	}

	ratings, err := h.userService.ListReceivedRatings(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, ratings)
}

func (h *UserHandler) GetFriendRequests(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	friendreqs, err := h.userService.GetFriendRequests(r.Context(), requesterID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, friendreqs)
}

func (h *UserHandler) SendFriendRequest(w http.ResponseWriter, r *http.Request) {
	targetid := r.PathValue("id")
	if targetid == "" {
		response.WriteError(w, http.StatusBadRequest, "missing user id")
		return
	}

	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	friendreq, err := h.userService.SendFriendRequest(r.Context(), requesterID, targetid)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, friendreq)
}

func (h *UserHandler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	targetid := r.PathValue("id")
	if targetid == "" {
		response.WriteError(w, http.StatusBadRequest, "missing user id")
		return
	}

	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	datetmp := r.PathValue("date")
	if datetmp == "" {
		response.WriteError(w, http.StatusBadRequest, "missing request date")
		return
	}

	date, err := time.Parse("2006-01-02T15:04:05.999999Z", datetmp)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request date")
		return
	}

	friendreq, err := h.userService.AcceptFriendRequest(r.Context(), requesterID, targetid, date)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, friendreq)
}

func (h *UserHandler) DenyFriendRequest(w http.ResponseWriter, r *http.Request) {
	targetid := r.PathValue("id")
	if targetid == "" {
		response.WriteError(w, http.StatusBadRequest, "missing user id")
		return
	}

	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	datetmp := r.PathValue("date")
	if datetmp == "" {
		response.WriteError(w, http.StatusBadRequest, "missing request date")
		return
	}

	date, err := time.Parse("2006-01-02T15:04:05.999999Z", datetmp)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request date")
		return
	}

	friendreq, err := h.userService.DenyFriendRequest(r.Context(), requesterID, targetid, date)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, friendreq)
}

func (h *UserHandler) Unfriend(w http.ResponseWriter, r *http.Request) {
	targetid := r.PathValue("id")
	if targetid == "" {
		response.WriteError(w, http.StatusBadRequest, "missing user id")
		return
	}

	requesterID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	err := h.userService.Unfriend(r.Context(), requesterID, targetid)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	type resp struct {
		Text string `json:"message"`
	}

	response.WriteJSON(w, http.StatusOK, resp{Text: fmt.Sprintf("friendship with %s deleted successfully", targetid)})
}

func (h *UserHandler) GetUserFriends(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.WriteError(w, http.StatusBadRequest, "missing user id")
		return
	}
	if id == "me" {
		requesterID, ok := requester.ID(w, r, h.resolver)
		if !ok {
			return
		}
		id = requesterID
	}

	friends, err := h.userService.GetUserFriends(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, friends)
}
