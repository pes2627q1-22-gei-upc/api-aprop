package assistant_service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"api-aprop/internal/clients/groq_client"
	"api-aprop/internal/models/chatbot_model"
	"api-aprop/internal/repositories/catalogue_repository"
	"api-aprop/internal/repositories/chatbot_repository"
	"api-aprop/internal/repositories/preference_repository"
)

const (
	// ReplyTimeout leaves margin under the 10 s response time requirement.
	ReplyTimeout = 8 * time.Second

	MaxMessageRunes = 1000
	maxToolRounds   = 4
	historyMessages = 10
	// topicHistory is enough to understand follow ups like "i demà?".
	topicHistory = 2
)

type AssistantService struct {
	chatbotRepository   chatbot_repository.Repository
	catalogueRepository catalogue_repository.Repository
	ai                  groq_client.Client
	now                 func() time.Time
	replyTimeout        time.Duration
	preferences         preference_repository.Repository // optional, see WithPreferences
	limiter             rateLimiter
}

// WithPreferences lets the help answers use the state of the user, such as having done the interest test.
func (s *AssistantService) WithPreferences(r preference_repository.Repository) *AssistantService {
	s.preferences = r
	return s
}

// NewAssistantService builds the service. now must return the time in the local time zone of the users,
// since it is used to resolve relative dates.
func NewAssistantService(
	chatbotRepository chatbot_repository.Repository,
	catalogueRepository catalogue_repository.Repository,
	ai groq_client.Client,
	now func() time.Time,
	replyTimeout time.Duration,
) *AssistantService {
	return &AssistantService{
		chatbotRepository:   chatbotRepository,
		catalogueRepository: catalogueRepository,
		ai:                  ai,
		now:                 now,
		replyTimeout:        replyTimeout,
	}
}

// SendMessage asks the chatbot and, only if it answers, stores both messages.
// Nothing is stored when the chatbot fails, so the client can retry the same message.
func (s *AssistantService) SendMessage(ctx context.Context, userID string, text string, byVoice bool) (chatbot_model.Exchange, error) {
	userID, err := normalizeUserID(userID)
	if err != nil {
		return chatbot_model.Exchange{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > MaxMessageRunes {
		return chatbot_model.Exchange{}, fmt.Errorf("%w: the text must have between 1 and %d characters", ErrInvalidMessage, MaxMessageRunes)
	}

	now := s.now()
	if !s.limiter.allow(userID, now) {
		return chatbot_model.Exchange{}, ErrTooManyMessages
	}
	userMessage :=chatbot_model.Message{SentAt: timestamp(now), Sender: chatbot_model.SenderUser, Text: text, ByVoice: byVoice}

	// TODO(DP-017): a conversation never ends; every message goes to the latest one.
	startedAt, err := s.chatbotRepository.LatestConversation(ctx, userID)
	var history []chatbot_model.Message
	switch {
	case errors.Is(err, chatbot_repository.ErrConversationNotFound):
		startedAt = userMessage.SentAt
	case err != nil:
		return chatbot_model.Exchange{}, err
	default:
		if history, err = s.chatbotRepository.ListMessages(ctx, userID, startedAt); err != nil {
			return chatbot_model.Exchange{}, err
		}
	}

	replyCtx, cancel := context.WithTimeout(ctx, s.replyTimeout)
	defer cancel()
	assistantMessage, err := s.reply(replyCtx, userID, now, history, text)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return chatbot_model.Exchange{}, fmt.Errorf("%w: %w", ErrAssistantTimeout, err)
		}
		return chatbot_model.Exchange{}, fmt.Errorf("%w: %w", ErrAssistantUnavailable, err)
	}
	assistantMessage.SentAt = timestamp(s.now())
	if !assistantMessage.SentAt.After(userMessage.SentAt) {
		assistantMessage.SentAt = userMessage.SentAt.Add(time.Microsecond)
	}

	for _, m := range []chatbot_model.Message{userMessage, assistantMessage} {
		if err := validateMessage(m); err != nil {
			return chatbot_model.Exchange{}, err
		}
	}
	if err := s.chatbotRepository.SaveMessages(ctx, userID, startedAt, userMessage, assistantMessage); err != nil {
		return chatbot_model.Exchange{}, err
	}
	return chatbot_model.Exchange{
		ConversationStartedAt: startedAt,
		UserMessage:           userMessage,
		AssistantMessage:      assistantMessage,
	}, nil
}

// SendFeedback stores whether an assistant message was helpful. The message is identified by the
// start of its conversation and its own sent time, as returned when it was sent.
func (s *AssistantService) SendFeedback(ctx context.Context, userID string, conversationStartedAt, sentAt time.Time, helpful bool) error {
	userID, err := normalizeUserID(userID)
	if err != nil {
		return err
	}
	err = s.chatbotRepository.SaveFeedback(ctx, userID, timestamp(conversationStartedAt), timestamp(sentAt), helpful)
	if errors.Is(err, chatbot_repository.ErrMessageNotFound) {
		return fmt.Errorf("%w: %w", ErrMessageNotFound, err)
	}
	return err
}

func (s *AssistantService) GetConversation(ctx context.Context, userID string) (chatbot_model.Conversation, error) {
	userID, err := normalizeUserID(userID)
	if err != nil {
		return chatbot_model.Conversation{}, err
	}
	startedAt, err := s.chatbotRepository.LatestConversation(ctx, userID)
	if errors.Is(err, chatbot_repository.ErrConversationNotFound) {
		return chatbot_model.Conversation{}, fmt.Errorf("%w: %w", ErrConversationNotFound, err)
	}
	if err != nil {
		return chatbot_model.Conversation{}, err
	}
	messages, err := s.chatbotRepository.ListMessages(ctx, userID, startedAt)
	if err != nil {
		return chatbot_model.Conversation{}, err
	}
	return chatbot_model.Conversation{StartedAt: startedAt, Messages: messages}, nil
}

// lastSearch is the latest successful search_events call of a reply.
type lastSearch struct {
	input    SearchEventsInput
	eventIDs []int
}

// reply runs the agent loop: Groq decides which tools to call until it answers with text.
// Messages that are not about the app get a fixed reply and never reach the tools.
func (s *AssistantService) reply(ctx context.Context, userID string, now time.Time, history []chatbot_model.Message, text string) (chatbot_model.Message, error) {
	onTopic, err := s.isOnTopic(ctx, history, text)
	if err != nil {
		return chatbot_model.Message{}, err
	}
	if !onTopic {
		log.Printf("assistant: off-topic message")
		return chatbot_model.Message{Sender: chatbot_model.SenderAssistant, Text: offTopicReply}, nil
	}

	messages := make([]groq_client.Message, 0, len(history)+2)
	messages = append(messages, groq_client.SystemMessage(systemInstruction(now)))
	messages = append(messages, historyToMessages(history)...)
	messages = append(messages, groq_client.UserMessage(text))

	groqTools := toolDefinitions()

	var search *lastSearch
	var action string
	for range maxToolRounds {
		resp, err := s.ai.ChatCompletion(ctx, groq_client.ChatCompletionRequest{
			Messages: messages,
			Tools:    groqTools,
		})
		if err != nil {
			return chatbot_model.Message{}, err
		}
		if len(resp.Choices) == 0 {
			return chatbot_model.Message{}, errors.New("empty choices from groq")
		}

		assistantMsg := resp.Choices[0].Message
		messages = append(messages, assistantMsg)

		if len(assistantMsg.ToolCalls) == 0 {
			replyText := stripThinking(assistantMsg.Content)
			if replyText == "" {
				return chatbot_model.Message{}, errors.New("empty reply from groq")
			}
			m, err := s.assistantMessage(ctx, replyText, search)
			m.Action = action
			return m, err
		}

		for _, call := range assistantMsg.ToolCalls {
			resultText, found, err := s.callTool(ctx, userID, now, call.Function.Name, call.Function.Arguments)
			if err != nil {
				return chatbot_model.Message{}, err
			}
			log.Printf("assistant: tool=%s arguments=%s", call.Function.Name, call.Function.Arguments)
			if call.Function.Name == ToolGetAppHelp {
				if screen := helpScreen(call.Function.Arguments); screen != "" {
					action = screen
				}
			}
			if found != nil {
				search = found
			}
			messages = append(messages, groq_client.ToolResultMessage(call.ID, resultText))
		}
	}
	return chatbot_model.Message{}, fmt.Errorf("no reply after %d tool rounds", maxToolRounds)
}

// isOnTopic asks the model, without tools, whether the message is about the app.
// An unexpected answer counts as on topic: the main prompt also refuses unrelated questions.
func (s *AssistantService) isOnTopic(ctx context.Context, history []chatbot_model.Message, text string) (bool, error) {
	if len(history) > topicHistory {
		history = history[len(history)-topicHistory:]
	}
	messages := make([]groq_client.Message, 0, len(history)+2)
	messages = append(messages, groq_client.SystemMessage(topicInstruction))
	messages = append(messages, historyToMessages(history)...)
	messages = append(messages, groq_client.UserMessage(text))

	resp, err := s.ai.ChatCompletion(ctx, groq_client.ChatCompletionRequest{
		Messages: messages,
	})
	if err != nil {
		return false, fmt.Errorf("classify topic: %w", err)
	}
	if len(resp.Choices) == 0 {
		return true, nil
	}
	return !strings.Contains(strings.ToLower(resp.Choices[0].Message.Content), labelOffTopic), nil
}

// assistantMessage builds the reply. The interpretation is what the model searched for and the
// recommendations are the events the catalogue returned, so the model cannot invent either of them.
func (s *AssistantService) assistantMessage(ctx context.Context, text string, search *lastSearch) (chatbot_model.Message, error) {
	m := chatbot_model.Message{Sender: chatbot_model.SenderAssistant, Text: text}
	if search == nil {
		return m, nil
	}
	m.Interpretation = toInterpretation(search.input)
	events, err := s.catalogueRepository.ListByIDs(ctx, search.eventIDs)
	if err != nil {
		return chatbot_model.Message{}, err
	}
	m.RecommendedEvents = events
	return m, nil
}

func toInterpretation(in SearchEventsInput) *chatbot_model.Interpretation {
	return &chatbot_model.Interpretation{
		Category:     strings.TrimSpace(in.Category),
		Municipality: strings.TrimSpace(in.Municipality),
		DateFrom:     parseDate(in.From),
		DateTo:       parseDate(in.To),
	}
}

func parseDate(s string) *time.Time {
	t, err := time.Parse(DateLayout, s)
	if err != nil {
		return nil
	}
	return &t
}

// historyToMessages replays the latest messages so the model understands follow up questions.
func historyToMessages(history []chatbot_model.Message) []groq_client.Message {
	if len(history) > historyMessages {
		history = history[len(history)-historyMessages:]
	}
	msgs := make([]groq_client.Message, 0, len(history))
	for _, m := range history {
		if m.Sender == chatbot_model.SenderUser {
			msgs = append(msgs, groq_client.UserMessage(m.Text))
		} else {
			msgs = append(msgs, groq_client.AssistantMessage(m.Text))
		}
	}
	return msgs
}

func stripThinking(text string) string {
	for {
		start := strings.Index(text, "<think>")
		if start == -1 {
			break
		}
		end := strings.Index(text, "</think>")
		if end == -1 {
			text = text[:start]
			break
		}
		text = text[:start] + text[end+len("</think>"):]
	}
	return strings.TrimSpace(text)
}

// timestamp drops what Postgres cannot store, so the stored and returned times are equal.
func timestamp(t time.Time) time.Time {
	return t.UTC().Truncate(time.Microsecond)
}

// normalizeUserID accepts only the canonical UUID form (8-4-4-4-12 hex digits) and lowercases it.
func normalizeUserID(userID string) (string, error) {
	if len(userID) != 36 {
		return "", fmt.Errorf("%w: %q", ErrInvalidUserID, userID)
	}
	for i, c := range userID {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return "", fmt.Errorf("%w: %q", ErrInvalidUserID, userID)
			}
		case !isHex(c):
			return "", fmt.Errorf("%w: %q", ErrInvalidUserID, userID)
		}
	}
	return strings.ToLower(userID), nil
}

func isHex(c rune) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// validateMessage checks the domain restriction R12.
func validateMessage(m chatbot_model.Message) error {
	switch m.Sender {
	case chatbot_model.SenderUser:
		if m.Interpretation != nil {
			return chatbot_model.ErrOnlyAssistantInterprets
		}
		if len(m.RecommendedEvents) > 0 || m.Action != "" {
			return chatbot_model.ErrOnlyAssistantRecommends
		}
	case chatbot_model.SenderAssistant:
		if m.ByVoice {
			return chatbot_model.ErrOnlyUserByVoice
		}
	default:
		return chatbot_model.ErrInvalidSender
	}
	return nil
}
