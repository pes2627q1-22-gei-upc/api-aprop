-- name: CreateChatbotConversation :exec
INSERT INTO ChatbotConversations (UserID, StartedAt)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(started_at)::timestamptz)
ON CONFLICT DO NOTHING;

-- name: GetLatestChatbotConversation :one
SELECT StartedAt FROM ChatbotConversations
WHERE UserID = sqlc.arg(user_id)::uuid
ORDER BY StartedAt DESC
LIMIT 1;

-- name: CreateChatbotMessage :exec
INSERT INTO ChatbotMessages (
    UserID, ConversationStartedAt, SentAt, Sender, Text, ByVoice, Category, Municipality, DateFrom, DateTo
) VALUES (
    sqlc.arg(user_id)::uuid, sqlc.arg(conversation_started_at)::timestamptz, sqlc.arg(sent_at)::timestamptz,
    sqlc.arg(sender)::varchar, sqlc.arg(text)::varchar, sqlc.arg(by_voice)::boolean,
    sqlc.narg(category)::varchar, sqlc.narg(municipality)::varchar, sqlc.narg(date_from)::date, sqlc.narg(date_to)::date
);

-- name: CreateChatbotRecommendation :exec
INSERT INTO ChatbotRecommendations (UserID, ConversationStartedAt, SentAt, Event, Position)
VALUES (
    sqlc.arg(user_id)::uuid, sqlc.arg(conversation_started_at)::timestamptz, sqlc.arg(sent_at)::timestamptz,
    sqlc.arg(event_id)::integer, sqlc.arg(position)::integer
);

-- name: ListChatbotMessages :many
SELECT SentAt, Sender, Text, ByVoice, Category, Municipality, DateFrom, DateTo
FROM ChatbotMessages
WHERE UserID = sqlc.arg(user_id)::uuid AND ConversationStartedAt = sqlc.arg(conversation_started_at)::timestamptz
ORDER BY SentAt;

-- name: ListChatbotRecommendations :many
SELECT r.SentAt, sqlc.embed(e)
FROM ChatbotRecommendations r
JOIN Events e ON e.ID = r.Event
WHERE r.UserID = sqlc.arg(user_id)::uuid AND r.ConversationStartedAt = sqlc.arg(conversation_started_at)::timestamptz
ORDER BY r.SentAt, r.Position;

-- name: UpsertChatbotFeedback :exec
INSERT INTO ChatbotFeedback (UserID, ConversationStartedAt, SentAt, Helpful)
VALUES ($1::uuid, $2::timestamptz, $3::timestamptz, $4::boolean)
ON CONFLICT (UserID, ConversationStartedAt, SentAt) DO UPDATE SET Helpful = EXCLUDED.Helpful, CreatedAt = now();
