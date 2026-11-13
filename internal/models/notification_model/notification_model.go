package notification_model

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Notification struct {
	ID               int              `json:"id"`
	UserID           pgtype.UUID      `json:"user-id"`
	NotificationDate pgtype.Timestamp `json:"notification-date"`
	Friend           pgtype.UUID      `json:"friend-id"`
	RequestDate      pgtype.Timestamp `json:"request-date"`
	Event            pgtype.Int4      `json:"event"`
	Type             int              `json:"type"`
	Read             bool             `json:"read"`
	Text             string           `json:"text"`
}

type NotificationType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
