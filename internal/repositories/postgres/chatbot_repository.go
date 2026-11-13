package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-aprop/internal/models/chatbot_model"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/user_model"
	"api-aprop/internal/repositories/chatbot_repository"
	"api-aprop/internal/repositories/postgres/sqlc"
)

// ChatbotRepository stores chatbot conversations in Postgres using the sqlc generated queries.
type ChatbotRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

var _ chatbot_repository.Repository = (*ChatbotRepository)(nil)

func NewChatbotRepository(pool *pgxpool.Pool) *ChatbotRepository {
	return &ChatbotRepository{pool: pool, queries: sqlc.New(pool)}
}

func (r *ChatbotRepository) LatestConversation(ctx context.Context, userID string) (time.Time, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return time.Time{}, err
	}
	startedAt, err := r.queries.GetLatestChatbotConversation(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, fmt.Errorf("latest conversation of user %s: %w", userID, chatbot_repository.ErrConversationNotFound)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("latest conversation of user %s: %w", userID, err)
	}
	return startedAt.Time, nil
}

func (r *ChatbotRepository) SaveMessages(ctx context.Context, userID string, conversationStartedAt time.Time, messages ...chatbot_model.Message) error {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return err
	}
	conversation := toTimestamptz(conversationStartedAt)

	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)
		if err := q.CreateChatbotConversation(ctx, sqlc.CreateChatbotConversationParams{UserID: uid, StartedAt: conversation}); err != nil {
			return fmt.Errorf("create conversation: %w", err)
		}
		for _, m := range messages {
			if err := q.CreateChatbotMessage(ctx, toChatbotMessageParams(uid, conversation, m)); err != nil {
				return fmt.Errorf("create message: %w", err)
			}
			for position, e := range m.RecommendedEvents {
				err := q.CreateChatbotRecommendation(ctx, sqlc.CreateChatbotRecommendationParams{
					UserID:                uid,
					ConversationStartedAt: conversation,
					SentAt:                toTimestamptz(m.SentAt),
					EventID:               int32(e.ID),
					Position:              int32(position),
				})
				if err != nil {
					return fmt.Errorf("recommend event %d: %w", e.ID, err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("save chatbot messages of user %s: %w", userID, err)
	}
	return nil
}

func (r *ChatbotRepository) ListMessages(ctx context.Context, userID string, conversationStartedAt time.Time) ([]chatbot_model.Message, error) {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return nil, err
	}
	conversation := toTimestamptz(conversationStartedAt)

	rows, err := r.queries.ListChatbotMessages(ctx, sqlc.ListChatbotMessagesParams{UserID: uid, ConversationStartedAt: conversation})
	if err != nil {
		return nil, fmt.Errorf("list chatbot messages of user %s: %w", userID, err)
	}
	recommendations, err := r.queries.ListChatbotRecommendations(ctx, sqlc.ListChatbotRecommendationsParams{UserID: uid, ConversationStartedAt: conversation})
	if err != nil {
		return nil, fmt.Errorf("list chatbot recommendations of user %s: %w", userID, err)
	}
	eventsBySentAt := make(map[time.Time][]event_model.Event)
	for _, rec := range recommendations {
		sentAt := rec.Sentat.Time.UTC()
		eventsBySentAt[sentAt] = append(eventsBySentAt[sentAt], toModel(rec.Event, 0))
	}

	messages := make([]chatbot_model.Message, 0, len(rows))
	for _, row := range rows {
		m := chatbot_model.Message{
			SentAt:  row.Sentat.Time.UTC(),
			Sender:  chatbot_model.Sender(row.Sender),
			Text:    row.Text,
			ByVoice: row.Byvoice,
		}
		if m.Sender == chatbot_model.SenderAssistant {
			m.Interpretation = &chatbot_model.Interpretation{
				Category:     row.Category.String,
				Municipality: row.Municipality.String,
				DateFrom:     fromDate(row.Datefrom),
				DateTo:       fromDate(row.Dateto),
			}
			m.RecommendedEvents = eventsBySentAt[m.SentAt]
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (r *ChatbotRepository) SaveFeedback(ctx context.Context, userID string, conversationStartedAt, sentAt time.Time, helpful bool) error {
	uid, err := user_model.ParseID(userID)
	if err != nil {
		return err
	}
	err = r.queries.UpsertChatbotFeedback(ctx, sqlc.UpsertChatbotFeedbackParams{
		UserID:                uid,
		ConversationStartedAt: toTimestamptz(conversationStartedAt),
		SentAt:                toTimestamptz(sentAt),
		Helpful:               helpful,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
		return fmt.Errorf("feedback of user %s: %w", userID, chatbot_repository.ErrMessageNotFound)
	}
	if err != nil {
		return fmt.Errorf("save chatbot feedback of user %s: %w", userID, err)
	}
	return nil
}

func toChatbotMessageParams(uid pgtype.UUID, conversation pgtype.Timestamptz, m chatbot_model.Message) sqlc.CreateChatbotMessageParams {
	params := sqlc.CreateChatbotMessageParams{
		UserID:                uid,
		ConversationStartedAt: conversation,
		SentAt:                toTimestamptz(m.SentAt),
		Sender:                string(m.Sender),
		Text:                  m.Text,
		ByVoice:               m.ByVoice,
	}
	if i := m.Interpretation; i != nil {
		params.Category = toText(i.Category)
		params.Municipality = toText(i.Municipality)
		if i.DateFrom != nil {
			params.DateFrom = toDate(*i.DateFrom)
		}
		if i.DateTo != nil {
			params.DateTo = toDate(*i.DateTo)
		}
	}
	return params
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// fromDate returns nil for NULL dates.
func fromDate(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}
