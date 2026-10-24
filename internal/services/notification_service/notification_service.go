package notification_service

import (
	"errors"
	"fmt"

	"api-aprop/internal/models/notification_model"
	"api-aprop/internal/repositories/notification_repository"
)

type NotificationService struct {
	notificationRepository notification_repository.Repository
}

func NewNotificationService(notificationRepository notification_repository.Repository) *NotificationService {
	return &NotificationService{
		notificationRepository: notificationRepository,
	}
}

func (s *NotificationService) GetNotification(id int) (notification_model.Notification, error) {
	n, err := s.notificationRepository.Get(id)
	if errors.Is(err, notification_repository.ErrNotificationNotFound) {
		return notification_model.Notification{}, fmt.Errorf("%w: %w", notification_repository.ErrNotificationNotFound, err)
	}
	if err != nil {
		return notification_model.Notification{}, err
	}
	return n, nil
}

func (s *NotificationService) GetNotificationTypes() ([]notification_model.NotificationType, error) {
	n, err := s.notificationRepository.GetNotificationTypes()
	if err != nil {
		return []notification_model.NotificationType{}, err
	}
	return n, nil
}

func (s *NotificationService) GetUserNotifications(id int) ([]notification_model.Notification, error) {
	notifs, err := s.notificationRepository.GetUserNotifications(id)
	if errors.Is(err, notification_repository.ErrNotificationNotFound) {
		return []notification_model.Notification{}, fmt.Errorf("%w: %w", notification_repository.ErrUserNotFound, err)
	}
	if err != nil {
		return []notification_model.Notification{}, err
	}
	return notifs, nil
}

func (s *NotificationService) CreateNotification(n notification_model.Notification) (int, error) {
	id, err := s.notificationRepository.Create(n)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *NotificationService) ReadNotification(id int) (error) {
	err := s.notificationRepository.ReadNotification(id)
	if err != nil {
		return err
	}
	return nil
}

func (s *NotificationService) ReadUserNotifications(id int) (error) {
	err := s.notificationRepository.ReadUserNotifications(id)
	if err != nil {
		return err
	}
	return nil
}