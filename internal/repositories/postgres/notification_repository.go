package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"

	"api-aprop/internal/models/notification_model"
	"api-aprop/internal/repositories/notification_repository"
	"api-aprop/internal/repositories/postgres/sqlc"
)

type NotificationRepository struct {
	queries *sqlc.Queries
}

var _ notification_repository.Repository = (*NotificationRepository)(nil)

const NumNotificationTypes int = 3

func (r *NotificationRepository) loadNotificationTypes() (error){
	rows, err := r.queries.GetNotificationTypes(context.Background())
	if err!=nil{
		return err
	}
	if len(rows)!=NumNotificationTypes {
		err := r.queries.RemoveNotificationTypes(context.Background())
		if err!=nil{
			return err
		}
		err = r.queries.RestoreNotificationTypes(context.Background())
		if err!=nil{
			return err
		}
	}
	return nil
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	repo := &NotificationRepository{queries: sqlc.New(pool)}
	err:=repo.loadNotificationTypes()
	if err!=nil{
		panic("failed to load notification types")
	}
	return repo
}

func (r *NotificationRepository) Get(id int) (notification_model.Notification, error) {
	row, err := r.queries.GetNotification(context.Background(), int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return notification_model.Notification{}, fmt.Errorf("get notification %d: %w", id, notification_repository.ErrNotificationNotFound)
	}
	if err != nil {
		return notification_model.Notification{}, fmt.Errorf("get notification %d: %w", id, err)
	}

	return toNotification(row), nil
}

func (r *NotificationRepository) GetNotificationTypes() ([]notification_model.NotificationType, error){
	rows, err := r.queries.GetNotificationTypes(context.Background())
	if err!=nil{
		return []notification_model.NotificationType{}, err
	}
	nottypes := []notification_model.NotificationType{}
	for _, row := range(rows){
		nottypes=append(nottypes, notification_model.NotificationType{ID: int(row.ID), Name: row.Name})
	}
	return nottypes, nil
}

func (r *NotificationRepository) Create(n notification_model.Notification) (int, error) {
	row, err := r.queries.CreateNotification(context.Background(), sqlc.CreateNotificationParams{
		Userid: n.UserID,
		Notificationdate: n.NotificationDate,
		Requestdate: n.RequestDate,
		Event: n.Event,
		Type: int32(n.Type),
		Friend: n.Friend,
	})
	if err != nil {
		return 0, fmt.Errorf("create notification: %w", err)
	}

	return int(row), nil
}

func (r *NotificationRepository) GetUserNotifications(id pgtype.UUID, params map[string]any) ([]notification_model.Notification, error) {
	typefiltertmp := params["filterType"].(string)
	var typefilter int
	var err error
	if typefiltertmp == ""{
		typefilter=0
	} else {
		typefilter, err=strconv.Atoi(typefiltertmp)
		if err!=nil{
			return []notification_model.Notification{}, err
		}
	}

	rows, err := r.queries.GetUserNotifications(context.Background(), sqlc.GetUserNotificationsParams{Userid: id, Filtertype: int32(typefilter)})
	if errors.Is(err, pgx.ErrNoRows) {
		return []notification_model.Notification{}, fmt.Errorf("get notifications from user %d: %w", id, notification_repository.ErrUserNotFound)
	}
	if err != nil {
		return []notification_model.Notification{}, fmt.Errorf("get notification from user %d: %w", id, err)
	}

	notifs := []notification_model.Notification{}
	for _, row := range(rows){
		notifs=append(notifs, toNotification(row))
	}

	return notifs, nil
}

func (r *NotificationRepository) ReadNotification(id int) (error) {
	err := r.queries.ReadNotification(context.Background(), int32(id))
	if err != nil {
		return fmt.Errorf("read notification %d: %w", id, err)
	}

	return nil
}

func (r *NotificationRepository) ReadUserNotifications(id pgtype.UUID) (error) {
	err := r.queries.ReadUserNotifications(context.Background(), id)
	if err != nil {
		return fmt.Errorf("read notifications from user %d: %w", id, err)
	}

	return nil
}

func toNotification(row sqlc.Notification) (notification_model.Notification){
	return notification_model.Notification{
		ID: int(row.ID),
		NotificationDate: row.Notificationdate,
		RequestDate: row.Requestdate,
		Event: row.Event,
		Type: int(row.Type),
		Read: row.Read,
		UserID:	row.Userid,
		Friend: row.Friend,
	}
}