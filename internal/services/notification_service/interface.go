package notification_service

import (

	"api-aprop/internal/models/notification_model"
)

// Service contains the business logic for events.
type Service interface {
	GetNotification(id int) (notification_model.Notification, error)
	GetUserNotifications(id int, params map[string]any) ([]notification_model.Notification, error)
	CreateNotification(n notification_model.Notification) (int, error)
	ReadNotification(id int) (error)
	ReadUserNotifications(id int) (error)
	GetNotificationTypes() ([]notification_model.NotificationType, error)
}