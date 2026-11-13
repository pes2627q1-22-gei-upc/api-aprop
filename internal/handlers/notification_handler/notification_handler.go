package notification_handler

import (
	"fmt"
	"net/http"
	"strconv"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/services/notification_service"
)

type NotificationHandler struct {
	notificationService notification_service.Service
}

func NewNotificationHandler(notificationService notification_service.Service) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

func (h *NotificationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /notifications/{id}", h.GetNotification)
	mux.HandleFunc("GET /notifications/user/{id}", h.GetUserNotifications)
	mux.HandleFunc("POST /notifications/{id}/read", h.ReadNotification)
	mux.HandleFunc("POST /notifications/user/{id}/read", h.ReadUserNotifications)
	mux.HandleFunc("GET /notificationtypes", h.GetNotificationTypes)
}

func (h *NotificationHandler) GetNotification(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid notification id")
		return
	}

	notif, err := h.notificationService.GetNotification(id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, notif)
}

func (h *NotificationHandler) GetNotificationTypes(w http.ResponseWriter, r *http.Request) {
	types, err := h.notificationService.GetNotificationTypes()
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, types)
}

// TODO(DP-011): {id} is not checked against the requester, so any signed in user can read (and mark as
// read) the notifications of another user. It should use requester.ID like the other handlers.
func (h *NotificationHandler) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	tmp := r.PathValue("id")
	id, err := user_model.ParseID(tmp)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	params := map[string]any{}
	params["filterType"] = r.URL.Query().Get("filterType")

	notifs, err := h.notificationService.GetUserNotifications(id, params)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, notifs)
}

func (h *NotificationHandler) ReadNotification(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid notification id")
		return
	}

	err = h.notificationService.ReadNotification(id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, struct {
		Message string `json:"message"`
	}{Message: fmt.Sprintf("notification %d read", id)})
}

func (h *NotificationHandler) ReadUserNotifications(w http.ResponseWriter, r *http.Request) {
	tmp := r.PathValue("id")
	id, err := user_model.ParseID(tmp)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	err = h.notificationService.ReadUserNotifications(id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, struct {
		Message string `json:"message"`
	}{Message: fmt.Sprintf("notifications of user %s read", id)})
}
