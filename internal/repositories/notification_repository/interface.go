package notification_repository

import (

	"api-aprop/internal/models/notification_model"
)

type Repository interface{
	Get(id int) (notification_model.Notification, error)
	Create(notification_model.Notification) (int, error)
	GetUserNotifications(id int) ([]notification_model.Notification, error)
	ReadNotification(id int) (error)
	ReadUserNotifications(id int) (error)
	GetNotificationTypes() ([]notification_model.NotificationType, error)
}