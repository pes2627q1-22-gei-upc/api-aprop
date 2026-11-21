package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-aprop/internal/models/notification_model"
	"api-aprop/internal/repositories/notification_repository"
	"api-aprop/internal/repositories/postgres/sqlc"
)

type NotificationRepository struct {
	queries *sqlc.Queries
}

var _ notification_repository.Repository = (*NotificationRepository)(nil)

const NumNotificationTypes int = 4

func (r *NotificationRepository) loadNotificationTypes() error {
	rows, err := r.queries.GetNotificationTypes(context.Background())
	if err != nil {
		log.Printf("Err1")
		return err
	}
	if len(rows) != NumNotificationTypes {
		err := r.queries.RemoveNotificationTypes(context.Background())
		if err != nil {
			log.Printf("Err2")
			return err
		}
		err = r.queries.RestoreNotificationTypes(context.Background())
		if err != nil {
			log.Printf("Err3")
			return err
		}
	}
	return nil
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	repo := &NotificationRepository{queries: sqlc.New(pool)}
	err := repo.loadNotificationTypes()
	if err != nil {
		panic("failed to load notification types")
	}
	return repo
}

func (r *NotificationRepository) Get(id int) (notification_model.Notification, error) {
	row, err := r.queries.GetNotification(context.Background(), int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return notification_model.Notification{}, notification_repository.ErrNotificationNotFound
	}
	if err != nil {
		return notification_model.Notification{}, fmt.Errorf("get notification %d: %w", id, err)
	}

	return toNotification(row), nil
}

func (r *NotificationRepository) GetNotificationTypes() ([]notification_model.NotificationType, error) {
	rows, err := r.queries.GetNotificationTypes(context.Background())
	if err != nil {
		return []notification_model.NotificationType{}, err
	}
	nottypes := []notification_model.NotificationType{}
	for _, row := range rows {
		nottypes = append(nottypes, notification_model.NotificationType{ID: int(row.ID), Name: row.Name})
	}
	return nottypes, nil
}

func (r *NotificationRepository) Create(n notification_model.Notification) (int, error) {
	user, err := r.queries.GetUserByUUID(context.Background(), n.UserID)
	if err != nil {
		return 0, fmt.Errorf("fetch user: %w", err)
	}

	settings, err := r.queries.GetSettings(context.Background(), int32(user.Settings))
	if err != nil {
		return 0, fmt.Errorf("fetch settings: %w", err)
	}

	for _, v := range settings {
		if v.Notificationtype == int32(n.Type) {
			row, err := r.queries.CreateNotification(context.Background(), sqlc.CreateNotificationParams{
				Userid:      n.UserID,
				Requestdate: n.RequestDate,
				Event:       n.Event,
				Type:        int32(n.Type),
				Friend:      n.Friend,
				Text:        n.Text,
			})
			if err != nil {
				return 0, fmt.Errorf("create notification: %w", err)
			}

			return int(row), nil
		}
	}
	return 0, nil
}

func (r *NotificationRepository) GetUserNotifications(id pgtype.UUID, params map[string]any) ([]notification_model.Notification, error) {
	typefiltertmp := params["filterType"].(string)
	var typefilter int
	var err error
	if typefiltertmp == "" {
		typefilter = 0
	} else {
		typefilter, err = strconv.Atoi(typefiltertmp)
		if err != nil {
			return []notification_model.Notification{}, err
		}
	}

	_, err = r.queries.GetUserByUUID(context.Background(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		return []notification_model.Notification{}, notification_repository.ErrUserNotFound
	} else if err != nil {
		return []notification_model.Notification{}, fmt.Errorf("get notification from user %s: %w", id.String(), err)
	}

	rows, err := r.queries.GetUserNotifications(context.Background(), sqlc.GetUserNotificationsParams{Userid: id, Filtertype: int32(typefilter)})
	if err != nil {
		return []notification_model.Notification{}, fmt.Errorf("get notification from user %s: %w", id.String(), err)
	}

	notifs := []notification_model.Notification{}
	for _, row := range rows {
		notifs = append(notifs, toNotification(row))
	}

	return notifs, nil
}

func (r *NotificationRepository) ReadNotification(id int) error {
	err := r.queries.ReadNotification(context.Background(), int32(id))
	if err != nil {
		return fmt.Errorf("read notification %d: %w", id, err)
	}

	return nil
}

func (r *NotificationRepository) ReadUserNotifications(id pgtype.UUID) error {
	_, err := r.queries.GetUserByUUID(context.Background(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		return notification_repository.ErrUserNotFound
	} else if err != nil {
		return fmt.Errorf("get notification from user %s: %w", id.String(), err)
	}

	err = r.queries.ReadUserNotifications(context.Background(), id)
	if err != nil {
		return fmt.Errorf("read notifications from user %s: %w", id.String(), err)
	}

	return nil
}

func toNotification(row sqlc.Notification) notification_model.Notification {
	return notification_model.Notification{
		ID:               int(row.ID),
		NotificationDate: row.Notificationdate,
		RequestDate:      row.Requestdate,
		Event:            row.Event,
		Type:             int(row.Type),
		Read:             row.Read,
		UserID:           row.Userid,
		Friend:           row.Friend,
		Text:             row.Text,
	}
}

func (r *NotificationRepository) CreateReminders() ([]int32, error) {
	ids, err := r.queries.CreateReminders(context.Background())
	if err != nil {
		return []int32{}, fmt.Errorf("create reminders: %w", err)
	}

	return ids, nil
}

func (r *NotificationRepository) GetNotificationTypeByName(name string) (int, error) {
	id, err := r.queries.GetNotificationTypeByName(context.Background(), name)
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (r *NotificationRepository) CreateFriendSignupNotifications(id pgtype.UUID, event int, text string) ([]int32, error) {
	ids, err := r.queries.CreateFriendSignupNotifications(context.Background(), sqlc.CreateFriendSignupNotificationsParams{Event: int32(event), Userid: id, Signuptext: text})
	if err != nil {
		return []int32{}, err
	}

	return ids, nil
}
