package assistant_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"api-aprop/internal/clients/gemini_client"
	"api-aprop/internal/clients/mcp_client"
	"api-aprop/internal/mcp_server"
	"api-aprop/internal/models/chatbot_model"
	"api-aprop/internal/repositories/catalogue_repository"
	"api-aprop/internal/repositories/chatbot_repository"
)

const (
	// ReplyTimeout leaves margin under the 10 s response time requirement.
	ReplyTimeout = 8 * time.Second

	MaxMessageRunes = 1000
	maxToolRounds   = 4
	historyMessages = 10
)

type AssistantService struct {
	chatbotRepository   chatbot_repository.Repository
	catalogueRepository catalogue_repository.Repository
	gemini              gemini_client.Client
	mcp                 mcp_client.Client
	now                 func() time.Time
	replyTimeout        time.Duration
}

// NewAssistantService builds the service. now must return the time in the local time zone of the users,
// since it is used to resolve relative dates.
func NewAssistantService(
	chatbotRepository chatbot_repository.Repository,
	catalogueRepository catalogue_repository.Repository,
	gemini gemini_client.Client,
	mcp mcp_client.Client,
	now func() time.Time,
	replyTimeout time.Duration,
) *AssistantService {
	return &AssistantService{
		chatbotRepository:   chatbotRepository,
		catalogueRepository: catalogueRepository,
		gemini:              gemini,
		mcp:                 mcp,
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
	userMessage := chatbot_model.Message{SentAt: timestamp(now), Sender: chatbot_model.SenderUser, Text: text, ByVoice: byVoice}

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
	assistantMessage, err := s.reply(replyCtx, now, history, text)
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
		if err := m.Validate(); err != nil {
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
	input    mcp_server.SearchEventsInput
	eventIDs []int
}

// reply runs the agent loop: Gemini decides which MCP tools to call until it answers with text.
func (s *AssistantService) reply(ctx context.Context, now time.Time, history []chatbot_model.Message, text string) (chatbot_model.Message, error) {
	session, err := s.mcp.Connect(ctx)
	if err != nil {
		return chatbot_model.Message{}, err
	}
	defer session.Close()

	tools, err := session.ListTools(ctx)
	if err != nil {
		return chatbot_model.Message{}, err
	}
	request := gemini_client.InteractionRequest{
		SystemInstruction: systemInstruction(now),
		Input:             historySteps(history),
		Tools:             make([]gemini_client.FunctionTool, 0, len(tools)),
	}
	known := make(map[string]bool, len(tools))
	for _, t := range tools {
		known[t.Name] = true
		request.Tools = append(request.Tools, gemini_client.FunctionTool{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  gemini_client.SanitizeSchema(t.InputSchema),
		})
	}
	request.Input = append(request.Input, gemini_client.UserInputStep(text))

	var search *lastSearch
	for range maxToolRounds {
		interaction, err := s.gemini.CreateInteraction(ctx, request)
		if err != nil {
			return chatbot_model.Message{}, err
		}
		request.Input = append(request.Input, interaction.Steps...)

		calls := interaction.FunctionCalls()
		if len(calls) == 0 {
			replyText := interaction.OutputText()
			if replyText == "" {
				return chatbot_model.Message{}, errors.New("empty reply from gemini")
			}
			return s.assistantMessage(ctx, replyText, search)
		}
		for _, call := range calls {
			if !known[call.Name] {
				request.Input = append(request.Input, gemini_client.FunctionResultStep(call, "unknown tool "+call.Name, true))
				continue
			}
			result, err := session.CallTool(ctx, call.Name, call.Arguments)
			if err != nil {
				return chatbot_model.Message{}, err
			}
			if call.Name == mcp_server.ToolSearchEvents && !result.IsError {
				if found, ok := parseSearch(call.Arguments, result.Structured); ok {
					search = found
				}
			}
			request.Input = append(request.Input, gemini_client.FunctionResultStep(call, result.Text, result.IsError))
		}
	}
	return chatbot_model.Message{}, fmt.Errorf("no reply after %d tool rounds", maxToolRounds)
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

func parseSearch(arguments json.RawMessage, structured json.RawMessage) (*lastSearch, bool) {
	var input mcp_server.SearchEventsInput
	if len(arguments) > 0 && json.Unmarshal(arguments, &input) != nil {
		return nil, false
	}
	var output mcp_server.SearchEventsOutput
	if json.Unmarshal(structured, &output) != nil {
		return nil, false
	}
	ids := make([]int, 0, len(output.Events))
	for _, e := range output.Events {
		ids = append(ids, e.ID)
	}
	return &lastSearch{input: input, eventIDs: ids}, true
}

func toInterpretation(in mcp_server.SearchEventsInput) *chatbot_model.Interpretation {
	return &chatbot_model.Interpretation{
		Category:     strings.TrimSpace(in.Category),
		Municipality: strings.TrimSpace(in.Municipality),
		DateFrom:     parseDate(in.From),
		DateTo:       parseDate(in.To),
	}
}

func parseDate(s string) *time.Time {
	t, err := time.Parse(mcp_server.DateLayout, s)
	if err != nil {
		return nil
	}
	return &t
}

// historySteps replays the latest messages so the model understands follow up questions.
func historySteps(history []chatbot_model.Message) []json.RawMessage {
	if len(history) > historyMessages {
		history = history[len(history)-historyMessages:]
	}
	steps := make([]json.RawMessage, 0, len(history)+1)
	for _, m := range history {
		if m.Sender == chatbot_model.SenderUser {
			steps = append(steps, gemini_client.UserInputStep(m.Text))
		} else {
			steps = append(steps, gemini_client.ModelOutputStep(m.Text))
		}
	}
	return steps
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
