package notification_repository

import (

	"api-aprop/internal/models/notification_model"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository interface{
	Get(id int) (notification_model.Notification, error)
	Create(notification_model.Notification) (int, error)
	GetUserNotifications(id pgtype.UUID, params map[string]any) ([]notification_model.Notification, error)
	ReadNotification(id int) (error)
	ReadUserNotifications(id pgtype.UUID) (error)
	GetNotificationTypes() ([]notification_model.NotificationType, error)
}