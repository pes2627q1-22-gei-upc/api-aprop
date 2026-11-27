package rating_handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"api-aprop/internal/handlers/requester"
	"api-aprop/internal/handlers/response"
	"api-aprop/internal/services/rating_service"
)

type RatingHandler struct {
	service  rating_service.Service
	resolver requester.Resolver
}

func NewRatingHandler(service rating_service.Service, resolver requester.Resolver) *RatingHandler {
	return &RatingHandler{service: service, resolver: resolver}
}

// RegisterRoutes registers the rating endpoints.
func (h *RatingHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /events/{id}/ratings", h.Rate)
	mux.HandleFunc("PUT /events/{id}/ratings/me", h.UpdateMine)
	mux.HandleFunc("DELETE /events/{id}/ratings/me", h.DeleteMine)
	mux.HandleFunc("POST /events/{id}/creator/ratings", h.RateUser)
	mux.HandleFunc("PUT /events/{id}/creator/ratings/me", h.UpdateUserRating)
	mux.HandleFunc("DELETE /events/{id}/creator/ratings/me", h.DeleteUserRating)
	mux.HandleFunc("GET /events/{id}/enrollment/attendance", h.HasAttended)
	mux.HandleFunc("GET /events/{id}/signup/attendance", h.HasAttended)
}

// HasAttended handles GET /events/{id}/enrollment/attendance and GET /events/{id}/signup/attendance.
func (h *RatingHandler) HasAttended(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	status, err := h.service.HasAttended(r.Context(), userID, eventID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, status)
}

// Rate handles POST /events/{id}/ratings with body {"score": 1..5}.
func (h *RatingHandler) Rate(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}
	var body struct {
		Score *int `json:"score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Score == nil {
		response.WriteError(w, http.StatusBadRequest, `invalid body, expected {"score": 1..5}`)
		return
	}

	rating, err := h.service.Rate(r.Context(), userID, eventID, *body.Score)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, rating)
}

func (h *RatingHandler) RateUser(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}
	var body struct {
		Score *int `json:"score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Score == nil {
		response.WriteError(w, http.StatusBadRequest, `invalid body, expected {"score": 1..5}`)
		return
	}

	rating, err := h.service.RateUser(r.Context(), userID, eventID, *body.Score)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, rating)
}

func (h *RatingHandler) UpdateUserRating(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}
	var body struct {
		Score *int `json:"score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Score == nil {
		response.WriteError(w, http.StatusBadRequest, `invalid body, expected {"score": 1..5}`)
		return
	}

	rating, err := h.service.UpdateUserRating(r.Context(), userID, eventID, *body.Score)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, rating)
}

func (h *RatingHandler) DeleteUserRating(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}

	err = h.service.DeleteUserRating(r.Context(), userID, eventID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UpdateMine handles PUT /events/{id}/ratings/me with body {"score": 1..5}.
func (h *RatingHandler) UpdateMine(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}
	var body struct {
		Score *int `json:"score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Score == nil {
		response.WriteError(w, http.StatusBadRequest, `invalid body, expected {"score": 1..5}`)
		return
	}
	rating, err := h.service.UpdateRating(r.Context(), userID, eventID, *body.Score)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	// TODO(DP-036): 200 (POST answers 201) and the 404 for a missing rating are provisional.
	response.WriteJSON(w, http.StatusOK, rating)
}

// DeleteMine handles DELETE /events/{id}/ratings/me.
func (h *RatingHandler) DeleteMine(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}
	userID, ok := requester.ID(w, r, h.resolver)
	if !ok {
		return
	}
	if err := h.service.DeleteRating(r.Context(), userID, eventID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeServiceError maps rating service errors to HTTP responses.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, rating_service.ErrEventNotFound):
		response.WriteError(w, http.StatusNotFound, "event not found")
	case errors.Is(err, rating_service.ErrRatingNotFound):
		response.WriteError(w, http.StatusNotFound, "rating not found")
	case errors.Is(err, rating_service.ErrNotAttended):
		// TODO(DP-033): 403 chosen for R10 (alternatives 409/422); no frontend contract yet.
		response.WriteError(w, http.StatusForbidden, "user has not attended the event")
	case errors.Is(err, rating_service.ErrAlreadyRated):
		response.WriteError(w, http.StatusConflict, "user has already rated the event")
	case errors.Is(err, rating_service.ErrInvalidScore):
		response.WriteError(w, http.StatusBadRequest, "score must be between 1 and 5")
	case errors.Is(err, rating_service.ErrInvalidUserID):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, rating_service.ErrInvalidEventID):
		response.WriteError(w, http.StatusBadRequest, "invalid event id")
	case errors.Is(err, rating_service.ErrCreatorNotFound):
		response.WriteError(w, http.StatusNotFound, "creator not found")
	case errors.Is(err, rating_service.ErrUserAlreadyRated):
		response.WriteError(w, http.StatusConflict, "user has already rated the event's creator")
	case errors.Is(err, rating_service.ErrRateOwnEvent):
		response.WriteError(w, http.StatusBadRequest, "cannot rate own event")
	case errors.Is(err, rating_service.ErrRateSelf):
		response.WriteError(w, http.StatusBadRequest, "cannot rate yourself")
	case errors.Is(err, rating_service.ErrEventCanceled):
		response.WriteError(w, http.StatusBadRequest, "event has been canceled")
	default:
		log.Printf("internal error: %v", err)
		response.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
