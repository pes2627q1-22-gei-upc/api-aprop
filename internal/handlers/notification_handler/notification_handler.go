package notification_handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/models/notification_model"
	"api-aprop/internal/services/notification_service"
)

const maxBodyBytes = 1 << 20

const UserIDHeader = "X-User-ID"

type NotificationHandler struct {
	notificationService notification_service.Service
}

func NewNotificationHandler(notificationService notification_service.Service) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

func (h *NotificationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /notifications/{id}", h.GetNotification)
	mux.HandleFunc("GET /notifications/user/{id}", h.GetUserNotifications)
	mux.HandleFunc("POST /notifications", h.CreateNotification)
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

func (h *NotificationHandler) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	tmp := r.PathValue("id")
	id, err := uuid.FromString(tmp)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	params := map[string]any{}
	params["filterType"] = r.URL.Query().Get("filterType")

	notifs, err := h.notificationService.GetUserNotifications(toPGUUID(id), params)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, notifs)
}

type createNotificationRequest struct {
	UserID           string     `json:"user-id"`
	NotificationDate time.Time  `json:"notification-date"`
	Friend           string     `json:"friend"`
	RequestDate      *time.Time `json:"request-date"`
	Event            *int32     `json:"event"`
	Type             int        `json:"type"`
}

func (h *NotificationHandler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var n createNotificationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&n); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	id, err := uuid.FromString(n.UserID)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	requestdate := pgtype.Timestamp{}
	friend := pgtype.UUID{}
	event := pgtype.Int4{}

	notifdate := pgtype.Timestamp{Valid: true, Time: n.NotificationDate}
	if n.RequestDate != nil {
		requestdate = pgtype.Timestamp{Valid: true, Time: *n.RequestDate}
	}
	if n.Friend != "" {
		tmp, err := uuid.FromString(n.Friend)
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		friend = toPGUUID(tmp)
	}
	if n.Event != nil {
		event = pgtype.Int4{Valid: true, Int32: *n.Event}
	}

	notif := notification_model.Notification{
		UserID:           toPGUUID(id),
		NotificationDate: notifdate,
		Friend:           friend,
		RequestDate:      requestdate,
		Event:            event,
		Type:             n.Type,
	}

	created, err := h.notificationService.CreateNotification(notif)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, created)
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
	id, err := uuid.FromString(tmp)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	err = h.notificationService.ReadUserNotifications(toPGUUID(id))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, struct {
		Message string `json:"message"`
	}{Message: fmt.Sprintf("notifications of user %d read", id)})
}

func toPGUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Valid: true, Bytes: [16]byte(id)}
}
