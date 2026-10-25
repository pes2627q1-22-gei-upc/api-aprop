package notification_model

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Notification struct{
	ID					int					`json:"id"`
	UserID 				int					`json:"user-id"`
	NotificationDate	pgtype.Timestamp	`json:"notification-date"`
	Friend 				pgtype.Int4			`json:"friend-id"`
	RequestDate 		pgtype.Timestamp	`json:"request-date"`
	Event				pgtype.Int4			`json:"event"`
	Type				int					`json:"type"`
	Read				bool				`json:"read"`
}

type NotificationType struct{
	ID		int		`json:"id"`
	Name	string	`json:"name"`
}