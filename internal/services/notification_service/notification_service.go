package notification_service

import (
	"errors"

	"api-aprop/internal/models/notification_model"
	"api-aprop/internal/repositories/notification_repository"
	"github.com/jackc/pgx/v5/pgtype"
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
		return notification_model.Notification{}, ErrNotificationNotFound
	} else if err != nil {
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

func (s *NotificationService) GetUserNotifications(id pgtype.UUID, params map[string]any) ([]notification_model.Notification, error) {
	notifs, err := s.notificationRepository.GetUserNotifications(id, params)
	if errors.Is(err, notification_repository.ErrUserNotFound){
		return []notification_model.Notification{}, ErrUserNotFound
	} else if err != nil {
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

func (s *NotificationService) ReadNotification(id int) error {
	_, err := s.notificationRepository.Get(id)
	if errors.Is(err, notification_repository.ErrNotificationNotFound){
		return ErrNotificationNotFound
	} else if err!=nil{
		return err
	}
	
	err = s.notificationRepository.ReadNotification(id)
	if err != nil {
		return err
	}
	return nil
}

func (s *NotificationService) ReadUserNotifications(id pgtype.UUID) error {
	err := s.notificationRepository.ReadUserNotifications(id)
	if errors.Is(err, notification_repository.ErrUserNotFound){
		return ErrUserNotFound
	} else if err != nil {
		return err
	}
	return nil
}

func (s *NotificationService) CreateReminders() ([]int32, error) {
	ids, err := s.notificationRepository.CreateReminders()
	if err != nil {
		return []int32{}, err
	}
	return ids, nil
}
