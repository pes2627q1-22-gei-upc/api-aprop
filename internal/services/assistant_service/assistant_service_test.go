package assistant_service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"api-aprop/internal/clients/groq_client"
	"api-aprop/internal/clients/mcp_client"
	"api-aprop/internal/mcp_server"
	"api-aprop/internal/models/chatbot_model"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/repositories/catalogue_repository"
	"api-aprop/internal/repositories/chatbot_repository"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

var fixedNow = time.Date(2026, 10, 19, 16, 30, 0, 123456789, time.UTC)

// --- fakes ---

type fakeChatbotRepository struct {
	startedAt time.Time // zero: no conversation
	history   []chatbot_model.Message
	saveErr   error

	savedStartedAt time.Time
	saved          []chatbot_model.Message
}

func (f *fakeChatbotRepository) LatestConversation(context.Context, string) (time.Time, error) {
	if f.startedAt.IsZero() {
		return time.Time{}, chatbot_repository.ErrConversationNotFound
	}
	return f.startedAt, nil
}

func (f *fakeChatbotRepository) SaveMessages(_ context.Context, _ string, startedAt time.Time, messages ...chatbot_model.Message) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.savedStartedAt = startedAt
	f.saved = append(f.saved, messages...)
	return nil
}

func (f *fakeChatbotRepository) ListMessages(context.Context, string, time.Time) ([]chatbot_model.Message, error) {
	return f.history, nil
}

type fakeCatalogue struct{}

func (fakeCatalogue) Search(context.Context, catalogue_repository.SearchFilter) ([]event_model.Event, error) {
	return nil, nil
}

func (fakeCatalogue) ListByIDs(_ context.Context, ids []int) ([]event_model.Event, error) {
	events := make([]event_model.Event, 0, len(ids))
	for _, id := range ids {
		events = append(events, event_model.Event{ID: id, Title: "event"})
	}
	return events, nil
}

// fakeGroq returns the scripted responses in order and records the requests.
// Topic classification requests are answered with topic (on_topic by default) and recorded apart.
type fakeGroq struct {
	responses     []groq_client.ChatCompletionResponse
	topic         string
	err           error
	block         bool
	requests      []groq_client.ChatCompletionRequest
	topicRequests []groq_client.ChatCompletionRequest
}

func (f *fakeGroq) ChatCompletion(ctx context.Context, req groq_client.ChatCompletionRequest) (groq_client.ChatCompletionResponse, error) {
	req.Messages = append([]groq_client.Message(nil), req.Messages...)
	isTopic := len(req.Messages) > 0 && req.Messages[0].Content == topicInstruction
	if isTopic && f.err == nil && !f.block {
		f.topicRequests = append(f.topicRequests, req)
		topic := f.topic
		if topic == "" {
			topic = labelOnTopic
		}
		return textReply(topic), nil
	}
	f.requests = append(f.requests, req)
	if f.block {
		<-ctx.Done()
		return groq_client.ChatCompletionResponse{}, ctx.Err()
	}
	if f.err != nil {
		return groq_client.ChatCompletionResponse{}, f.err
	}
	if len(f.requests) > len(f.responses) {
		return groq_client.ChatCompletionResponse{}, errors.New("unexpected call")
	}
	return f.responses[len(f.requests)-1], nil
}

type fakeMCP struct {
	results map[string]mcp_client.ToolResult
	calls   []string
	closed  bool
}

func (f *fakeMCP) Connect(context.Context) (mcp_client.Session, error) { return f, nil }

func (f *fakeMCP) ListTools(context.Context) ([]mcp_client.Tool, error) {
	return []mcp_client.Tool{
		{Name: mcp_server.ToolSearchEvents, InputSchema: map[string]any{"type": "object", "additionalProperties": false}},
		{Name: mcp_server.ToolGetEvent, InputSchema: map[string]any{"type": "object"}},
	}, nil
}

func (f *fakeMCP) CallTool(_ context.Context, name string, args json.RawMessage) (mcp_client.ToolResult, error) {
	f.calls = append(f.calls, name+" "+string(args))
	return f.results[name], nil
}

func (f *fakeMCP) Close() error {
	f.closed = true
	return nil
}

// --- helpers ---

func functionCall(id, name, args string) groq_client.ChatCompletionResponse {
	return groq_client.ChatCompletionResponse{
		ID: "1",
		Choices: []groq_client.Choice{
			{
				Index: 0,
				Message: groq_client.Message{
					Role: "assistant",
					ToolCalls: []groq_client.ToolCall{
						{
							ID:   id,
							Type: "function",
							Function: groq_client.FunctionCall{
								Name:      name,
								Arguments: args,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
	}
}

func textReply(text string) groq_client.ChatCompletionResponse {
	return groq_client.ChatCompletionResponse{
		ID: "1",
		Choices: []groq_client.Choice{
			{
				Index: 0,
				Message: groq_client.Message{
					Role:    "assistant",
					Content: text,
				},
				FinishReason: "stop",
			},
		},
	}
}

func searchResult(ids ...int) mcp_client.ToolResult {
	out := mcp_server.SearchEventsOutput{}
	for _, id := range ids {
		out.Events = append(out.Events, mcp_server.EventSummary{ID: id})
	}
	b, _ := json.Marshal(out)
	return mcp_client.ToolResult{Text: string(b), Structured: b}
}

func newService(repo *fakeChatbotRepository, ai *fakeGroq, mcp *fakeMCP) *AssistantService {
	return NewAssistantService(repo, fakeCatalogue{}, ai, mcp, func() time.Time { return fixedNow }, ReplyTimeout)
}

func messageRoles(msgs []groq_client.Message) []string {
	roles := make([]string, len(msgs))
	for i, m := range msgs {
		roles[i] = m.Role
	}
	return roles
}

// --- tests ---

func TestSendMessage_SearchesAndRecommends(t *testing.T) {
	repo := &fakeChatbotRepository{}
	ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
		functionCall("call_1", mcp_server.ToolSearchEvents, `{"category":"concert","municipality":"Girona","from":"2026-10-24","to":"2026-10-25"}`),
		textReply("He entès: concerts a Girona aquest cap de setmana."),
	}}
	mcp := &fakeMCP{results: map[string]mcp_client.ToolResult{mcp_server.ToolSearchEvents: searchResult(7, 9)}}

	exchange, err := newService(repo, ai, mcp).SendMessage(context.Background(), strings.ToUpper(validUserID), "  concerts a Girona aquest cap de setmana ", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reply := exchange.AssistantMessage
	if reply.Sender != chatbot_model.SenderAssistant || reply.Text != "He entès: concerts a Girona aquest cap de setmana." {
		t.Errorf("reply = %+v", reply)
	}
	i := reply.Interpretation
	if i == nil || i.Category != "concert" || i.Municipality != "Girona" ||
		i.DateFrom.Format("2006-01-02") != "2026-10-24" || i.DateTo.Format("2006-01-02") != "2026-10-25" {
		t.Errorf("interpretation = %+v", i)
	}
	if len(reply.RecommendedEvents) != 2 || reply.RecommendedEvents[0].ID != 7 || reply.RecommendedEvents[1].ID != 9 {
		t.Errorf("recommended = %+v", reply.RecommendedEvents)
	}

	if exchange.UserMessage.Text != "concerts a Girona aquest cap de setmana" || exchange.UserMessage.SentAt.Nanosecond()%1000 != 0 {
		t.Errorf("user message = %+v", exchange.UserMessage)
	}
	if !reply.SentAt.After(exchange.UserMessage.SentAt) {
		t.Errorf("the reply must be sent after the user message: %v, %v", reply.SentAt, exchange.UserMessage.SentAt)
	}
	if !exchange.ConversationStartedAt.Equal(exchange.UserMessage.SentAt) || !repo.savedStartedAt.Equal(exchange.ConversationStartedAt) {
		t.Errorf("a new conversation starts with the first message")
	}
	if len(repo.saved) != 2 || repo.saved[0].Sender != chatbot_model.SenderUser || repo.saved[1].Sender != chatbot_model.SenderAssistant {
		t.Errorf("saved = %+v", repo.saved)
	}

	// The second request includes the assistant's tool call and the tool execution result.
	if len(ai.requests) != 2 {
		t.Fatalf("ai calls = %d", len(ai.requests))
	}
	second := ai.requests[1].Messages
	if got := strings.Join(messageRoles(second), ","); got != "system,user,assistant,tool" {
		t.Errorf("second messages = %s", got)
	}
	if second[2].ToolCalls[0].ID != "call_1" || second[3].ToolCallID != "call_1" {
		t.Errorf("steps = %+v / %+v", second[2], second[3])
	}
	if !strings.Contains(ai.requests[0].Messages[0].Content, "dilluns 2026-10-19") {
		t.Errorf("the system instruction must give today's date")
	}
	if _, ok := ai.requests[0].Tools[0].Function.Parameters.(map[string]any)["additionalProperties"]; ok {
		t.Errorf("tool schemas must be sanitized")
	}
	if !mcp.closed {
		t.Errorf("the MCP session must be closed")
	}
}

func TestSendMessage_R12(t *testing.T) {
	repo := &fakeChatbotRepository{}
	ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
		functionCall("c", mcp_server.ToolSearchEvents, `{"municipality":"Reus"}`),
		textReply("Activitats a Reus."),
	}}
	mcp := &fakeMCP{results: map[string]mcp_client.ToolResult{mcp_server.ToolSearchEvents: searchResult(1)}}

	exchange, err := newService(repo, ai, mcp).SendMessage(context.Background(), validUserID, "què hi ha a Reus", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	user, reply := exchange.UserMessage, exchange.AssistantMessage
	if !user.ByVoice || user.Interpretation != nil || len(user.RecommendedEvents) != 0 {
		t.Errorf("TestR12: the user message cannot recommend and keeps by voice: %+v", user)
	}
	if reply.ByVoice || len(reply.RecommendedEvents) != 1 {
		t.Errorf("TestR12: only the chatbot recommends and it never speaks by voice: %+v", reply)
	}
}

func TestSendMessage_ReplyWithoutSearch(t *testing.T) {
	repo := &fakeChatbotRepository{}
	ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{textReply("Hola! Quina activitat busques?")}}

	exchange, err := newService(repo, ai, &fakeMCP{}).SendMessage(context.Background(), validUserID, "hola", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exchange.AssistantMessage.Interpretation != nil || len(exchange.AssistantMessage.RecommendedEvents) != 0 {
		t.Errorf("reply = %+v", exchange.AssistantMessage)
	}
}

func TestSendMessage_ContinuesConversation(t *testing.T) {
	startedAt := fixedNow.Add(-time.Hour).Truncate(time.Microsecond)
	repo := &fakeChatbotRepository{startedAt: startedAt, history: []chatbot_model.Message{
		{Sender: chatbot_model.SenderUser, Text: "teatre a Girona"},
		{Sender: chatbot_model.SenderAssistant, Text: "Aquí tens teatre a Girona."},
	}}
	ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{textReply("I demà també n'hi ha.")}}

	exchange, err := newService(repo, ai, &fakeMCP{}).SendMessage(context.Background(), validUserID, "i demà?", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exchange.ConversationStartedAt.Equal(startedAt) || !repo.savedStartedAt.Equal(startedAt) {
		t.Errorf("the message must go to the latest conversation")
	}
	if got := strings.Join(messageRoles(ai.requests[0].Messages), ","); got != "system,user,assistant,user" {
		t.Errorf("history = %s", got)
	}
}

func TestSendMessage_OffTopicGetsFixedReply(t *testing.T) {
	repo := &fakeChatbotRepository{}
	ai := &fakeGroq{topic: " Off_Topic\n"}
	mcp := &fakeMCP{}

	exchange, err := newService(repo, ai, mcp).SendMessage(context.Background(), validUserID, "Quin model ets?", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reply := exchange.AssistantMessage
	if reply.Text != offTopicReply || reply.Interpretation != nil || len(reply.RecommendedEvents) != 0 {
		t.Errorf("reply = %+v", reply)
	}
	if len(ai.requests) != 0 || len(mcp.calls) != 0 {
		t.Errorf("off topic messages must not reach the agent nor the tools")
	}
	if len(repo.saved) != 2 {
		t.Errorf("the off topic exchange is stored like any other: %+v", repo.saved)
	}
	if tools := ai.topicRequests[0].Tools; len(tools) != 0 {
		t.Errorf("the classifier must not get tools: %+v", tools)
	}
}

func TestSendMessage_TopicUsesRecentHistory(t *testing.T) {
	startedAt := fixedNow.Add(-time.Hour).Truncate(time.Microsecond)
	repo := &fakeChatbotRepository{startedAt: startedAt, history: []chatbot_model.Message{
		{Sender: chatbot_model.SenderUser, Text: "hola"},
		{Sender: chatbot_model.SenderAssistant, Text: "Hola! Què busques?"},
		{Sender: chatbot_model.SenderUser, Text: "teatre a Girona"},
		{Sender: chatbot_model.SenderAssistant, Text: "Aquí tens teatre a Girona."},
	}}
	ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{textReply("Demà també n'hi ha.")}}

	if _, err := newService(repo, ai, &fakeMCP{}).SendMessage(context.Background(), validUserID, "i demà?", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := ai.topicRequests[0].Messages
	if got := strings.Join(messageRoles(msgs), ","); got != "system,user,assistant,user" {
		t.Fatalf("classifier messages = %s", got)
	}
	if !strings.Contains(msgs[1].Content, "teatre a Girona") || !strings.Contains(msgs[3].Content, "i demà?") {
		t.Errorf("classifier messages = %+v", msgs)
	}
}

func TestSendMessage_UnknownToolIsReportedToTheModel(t *testing.T) {
	ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
		functionCall("c", "delete_everything", `{}`),
		textReply("No ho puc fer."),
	}}
	mcp := &fakeMCP{}

	if _, err := newService(&fakeChatbotRepository{}, ai, mcp).SendMessage(context.Background(), validUserID, "hola", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mcp.calls) != 0 {
		t.Errorf("unknown tools must not reach the MCP server: %v", mcp.calls)
	}
	last := ai.requests[1].Messages
	lastMsg := last[len(last)-1]
	if !strings.Contains(lastMsg.Content, `"is_error":true`) {
		t.Errorf("the model must get an error result: %s", lastMsg.Content)
	}
}

func TestSendMessage_Failures(t *testing.T) {
	loop := make([]groq_client.ChatCompletionResponse, maxToolRounds)
	for i := range loop {
		loop[i] = functionCall("c", mcp_server.ToolSearchEvents, `{}`)
	}
	tests := map[string]struct {
		ai   *fakeGroq
		want error
	}{
		"groq error":    {&fakeGroq{err: errors.New("quota")}, ErrAssistantUnavailable},
		"empty reply":   {&fakeGroq{responses: []groq_client.ChatCompletionResponse{{ID: "1", Choices: []groq_client.Choice{{Index: 0, Message: groq_client.Message{Role: "assistant"}}}}}}, ErrAssistantUnavailable},
		"endless tools": {&fakeGroq{responses: loop}, ErrAssistantUnavailable},
		"groq too slow": {&fakeGroq{block: true}, ErrAssistantTimeout},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := &fakeChatbotRepository{}
			mcp := &fakeMCP{results: map[string]mcp_client.ToolResult{mcp_server.ToolSearchEvents: searchResult()}}
			s := NewAssistantService(repo, fakeCatalogue{}, tt.ai, mcp, func() time.Time { return fixedNow }, 50*time.Millisecond)

			_, err := s.SendMessage(context.Background(), validUserID, "hola", false)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
			if len(repo.saved) != 0 {
				t.Errorf("nothing is stored when the chatbot fails, so the client can retry")
			}
		})
	}
}

func TestSendMessage_InvalidInput(t *testing.T) {
	tests := map[string]struct {
		userID, text string
		want         error
	}{
		"bad user id": {"not-a-uuid", "hola", ErrInvalidUserID},
		"blank text":  {validUserID, "   ", ErrInvalidMessage},
		"long text":   {validUserID, strings.Repeat("a", MaxMessageRunes+1), ErrInvalidMessage},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ai := &fakeGroq{}
			_, err := newService(&fakeChatbotRepository{}, ai, &fakeMCP{}).SendMessage(context.Background(), tt.userID, tt.text, false)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
			if len(ai.requests) != 0 || len(ai.topicRequests) != 0 {
				t.Errorf("ai must not be called")
			}
		})
	}
}

func TestGetConversation(t *testing.T) {
	s := newService(&fakeChatbotRepository{}, &fakeGroq{}, &fakeMCP{})
	if _, err := s.GetConversation(context.Background(), validUserID); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("error = %v, want ErrConversationNotFound", err)
	}

	startedAt := fixedNow.Truncate(time.Microsecond)
	history := []chatbot_model.Message{{Sender: chatbot_model.SenderUser, Text: "hola"}}
	s = newService(&fakeChatbotRepository{startedAt: startedAt, history: history}, &fakeGroq{}, &fakeMCP{})
	conversation, err := s.GetConversation(context.Background(), validUserID)
	if err != nil || !conversation.StartedAt.Equal(startedAt) || len(conversation.Messages) != 1 {
		t.Fatalf("conversation = %+v, err = %v", conversation, err)
	}
}

func TestMessageValidate_R12(t *testing.T) {
	event := []event_model.Event{{ID: 1}}
	tests := map[string]struct {
		m    chatbot_model.Message
		want error
	}{
		"user recommends": {chatbot_model.Message{Sender: chatbot_model.SenderUser, RecommendedEvents: event}, chatbot_model.ErrOnlyAssistantRecommends},
		"user interprets": {chatbot_model.Message{Sender: chatbot_model.SenderUser, Interpretation: &chatbot_model.Interpretation{}}, chatbot_model.ErrOnlyAssistantInterprets},
		"chatbot voice":   {chatbot_model.Message{Sender: chatbot_model.SenderAssistant, ByVoice: true}, chatbot_model.ErrOnlyUserByVoice},
		"unknown sender":  {chatbot_model.Message{Sender: "admin"}, chatbot_model.ErrInvalidSender},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := tt.m.Validate(); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}
