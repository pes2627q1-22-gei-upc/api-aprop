package assistant_service

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"api-aprop/internal/clients/groq_client"
	"api-aprop/internal/models/chatbot_model"
	"api-aprop/internal/models/event_model"
	"api-aprop/internal/repositories/catalogue_repository"
)

type titledFakeCatalogue struct {
	byID      map[int]event_model.Event
	searchIDs []int
	// optional per-call results for multi-search turns
	searchBatches [][]int
	searches      []catalogue_repository.SearchFilter
}

func (f *titledFakeCatalogue) Search(_ context.Context, filter catalogue_repository.SearchFilter) ([]event_model.Event, error) {
	callIdx := len(f.searches)
	f.searches = append(f.searches, filter)

	ids := f.searchIDs
	if callIdx < len(f.searchBatches) {
		ids = f.searchBatches[callIdx]
	}
	if filter.Limit > 0 && len(ids) > filter.Limit {
		ids = ids[:filter.Limit]
	}
	return f.ListByIDs(context.Background(), ids)
}

func (f *titledFakeCatalogue) ListByIDs(_ context.Context, ids []int) ([]event_model.Event, error) {
	out := make([]event_model.Event, 0, len(ids))
	for _, id := range ids {
		if e, ok := f.byID[id]; ok {
			out = append(out, e)
		}
	}
	return out, nil
}

func eventIDs(events []event_model.Event) []int {
	ids := make([]int, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	return ids
}

func TestParseRecommendedIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantText string
		wantIDs  []int
		wantTag  bool
	}{
		{
			name:     "standard tag on last line",
			input:    "Et recomano:\n- **Les parets criden**\n- **Clave Gaudí**\n\n[RECOMMENDED_IDS: 42, 17, 88]",
			wantText: "Et recomano:\n- **Les parets criden**\n- **Clave Gaudí**",
			wantIDs:  []int{42, 17, 88},
			wantTag:  true,
		},
		{
			name:     "explicit none tag",
			input:    "Quin tipus d'activitat et ve de gust i a quina ciutat?\n[RECOMMENDED_IDS: none]",
			wantText: "Quin tipus d'activitat et ve de gust i a quina ciutat?",
			wantIDs:  nil,
			wantTag:  true,
		},
		{
			name:     "empty bracket tag",
			input:    "No he trobat concerts gratuïts.\n[RECOMMENDED_IDS: ]",
			wantText: "No he trobat concerts gratuïts.",
			wantIDs:  nil,
			wantTag:  true,
		},
		{
			name:     "json array and markdown wrapping",
			input:    "Aquí tens opcions.\n`**[RECOMMENDED_IDS: [10, #20, 30]]**`.",
			wantText: "Aquí tens opcions.",
			wantIDs:  []int{10, 20, 30},
			wantTag:  true,
		},
		{
			name:     "standalone line without brackets",
			input:    "Aquí tens opcions.\nRECOMMENDED_IDS: 5, 3, 1",
			wantText: "Aquí tens opcions.",
			wantIDs:  []int{5, 3, 1},
			wantTag:  true,
		},
		{
			name:     "no tag at all",
			input:    "Hola! En què et puc ajudar?",
			wantText: "Hola! En què et puc ajudar?",
			wantIDs:  nil,
			wantTag:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotText, gotIDs, gotTag := parseRecommendedIDs(tc.input)
			if gotText != tc.wantText {
				t.Errorf("cleanText = %q, want %q", gotText, tc.wantText)
			}
			if !slices.Equal(gotIDs, tc.wantIDs) {
				t.Errorf("ids = %v, want %v", gotIDs, tc.wantIDs)
			}
			if gotTag != tc.wantTag {
				t.Errorf("hasTag = %v, want %v", gotTag, tc.wantTag)
			}
		})
	}
}

func TestSendMessage_VagueQueryAsksClarifyingQuestionsWithoutCards(t *testing.T) {
	repo := &fakeChatbotRepository{}
	ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
		textReply("Per recomanar-te millor, quin tipus de pla et ve de gust (música, teatre, exposicions...) i en quina ciutat o data?\n[RECOMMENDED_IDS: none]"),
	}}
	cat := &titledFakeCatalogue{
		byID:      map[int]event_model.Event{1: {ID: 1, Title: "Jazz al parc"}},
		searchIDs: []int{1},
	}

	s := NewAssistantService(repo, cat, ai, func() time.Time { return fixedNow }, ReplyTimeout)
	exchange, err := s.SendMessage(context.Background(), validUserID, "recomana'm plans", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reply := exchange.AssistantMessage
	if strings.Contains(reply.Text, "RECOMMENDED_IDS") {
		t.Errorf("internal tag must be stripped from reply: %q", reply.Text)
	}
	if len(cat.searches) != 0 {
		t.Errorf("vague query should not search catalogue: %+v", cat.searches)
	}
	if reply.Interpretation != nil || len(reply.RecommendedEvents) != 0 {
		t.Errorf("clarifying question must have no interpretation or cards: %+v", reply)
	}
	// Verify system prompt instructs clarifying questions and 3-event batching.
	sysPrompt := ai.requests[0].Messages[0].Content
	for _, snippet := range []string{"clarifying questions", "AT MOST 3 events", "[RECOMMENDED_IDS:"} {
		if !strings.Contains(sysPrompt, snippet) {
			t.Errorf("system prompt missing %q", snippet)
		}
	}
}

func TestSendMessage_CardsMatchExactChosenIDsAndOrder(t *testing.T) {
	repo := &fakeChatbotRepository{}
	cat := &titledFakeCatalogue{
		byID: map[int]event_model.Event{
			10: {ID: 10, Title: "Espai Vapor"},
			20: {ID: 20, Title: "Les parets criden"},
			30: {ID: 30, Title: "La invenció del temps"},
			40: {ID: 40, Title: "Exposició MACBA"},
			50: {ID: 50, Title: "Clave Gaudí"},
		},
		searchIDs: []int{10, 20, 30, 40, 50},
	}
	ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
		functionCall("c1", ToolSearchEvents, `{"category":"exposició","municipality":"Barcelona"}`),
		textReply("He entès: exposicions a Barcelona.\n- **Les parets criden**\n- **Exposició MACBA**\n- **Clave Gaudí**\n[RECOMMENDED_IDS: 20, 40, 50]"),
	}}

	s := NewAssistantService(repo, cat, ai, func() time.Time { return fixedNow }, ReplyTimeout)
	exchange, err := s.SendMessage(context.Background(), validUserID, "exposicions a Barcelona", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reply := exchange.AssistantMessage
	if strings.Contains(reply.Text, "RECOMMENDED_IDS") {
		t.Errorf("tag must not appear in visible text: %q", reply.Text)
	}
	if got := eventIDs(reply.RecommendedEvents); !slices.Equal(got, []int{20, 40, 50}) {
		t.Fatalf("recommended IDs = %v, want [20 40 50]", got)
	}
}

func TestSendMessage_CapsRecommendationsAtThree(t *testing.T) {
	cat := &titledFakeCatalogue{
		byID: map[int]event_model.Event{
			1: {ID: 1, Title: "Concert U"},
			2: {ID: 2, Title: "Concert Dos"},
			3: {ID: 3, Title: "Concert Tres"},
			4: {ID: 4, Title: "Concert Quatre"},
			5: {ID: 5, Title: "Concert Cinc"},
		},
		searchIDs: []int{1, 2, 3, 4, 5},
	}

	t.Run("explicit tag with 5 IDs capped to 3", func(t *testing.T) {
		ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
			functionCall("c1", ToolSearchEvents, `{"category":"concert","municipality":"Girona"}`),
			textReply("Aquí tens concerts.\n[RECOMMENDED_IDS: 5, 3, 1, 4, 2]"),
		}}
		s := NewAssistantService(&fakeChatbotRepository{}, cat, ai, func() time.Time { return fixedNow }, ReplyTimeout)
		exchange, err := s.SendMessage(context.Background(), validUserID, "concerts a Girona", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := eventIDs(exchange.AssistantMessage.RecommendedEvents); !slices.Equal(got, []int{5, 3, 1}) {
			t.Fatalf("recommended IDs = %v, want [5 3 1]", got)
		}
	})

	t.Run("no tag fallback capped to 3", func(t *testing.T) {
		ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
			functionCall("c1", ToolSearchEvents, `{"category":"concert","municipality":"Girona"}`),
			textReply("He entès: concerts a Girona."),
		}}
		s := NewAssistantService(&fakeChatbotRepository{}, cat, ai, func() time.Time { return fixedNow }, ReplyTimeout)
		exchange, err := s.SendMessage(context.Background(), validUserID, "concerts a Girona", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := eventIDs(exchange.AssistantMessage.RecommendedEvents); !slices.Equal(got, []int{1, 2, 3}) {
			t.Fatalf("recommended IDs = %v, want [1 2 3]", got)
		}
	})
}

func TestSendMessage_MultipleSearchesCombineCandidatesAndPreserveOrder(t *testing.T) {
	cat := &titledFakeCatalogue{
		byID: map[int]event_model.Event{
			11: {ID: 11, Title: "Jazz al parc"},
			12: {ID: 12, Title: "Rock a la plaça"},
			21: {ID: 21, Title: "Exposició MACBA"},
			22: {ID: 22, Title: "Clave Gaudí"},
		},
		searchBatches: [][]int{
			{11, 12},
			{21, 22},
		},
	}
	ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
		functionCall("c1", ToolSearchEvents, `{"category":"concert","municipality":"Barcelona"}`),
		functionCall("c2", ToolSearchEvents, `{"category":"exposició","municipality":"Barcelona"}`),
		textReply("Et recomano **Jazz al parc**, **Clave Gaudí** i **Exposició MACBA**.\n[RECOMMENDED_IDS: 11, 22, 21]"),
	}}

	s := NewAssistantService(&fakeChatbotRepository{}, cat, ai, func() time.Time { return fixedNow }, ReplyTimeout)
	exchange, err := s.SendMessage(context.Background(), validUserID, "concerts o exposicions a Barcelona", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := eventIDs(exchange.AssistantMessage.RecommendedEvents); !slices.Equal(got, []int{11, 22, 21}) {
		t.Fatalf("recommended IDs = %v, want [11 22 21]", got)
	}
}

func TestSendMessage_ExplicitNoneTagSuppressesCardsAndDropsHallucinatedIDs(t *testing.T) {
	cat := &titledFakeCatalogue{
		byID: map[int]event_model.Event{
			10: {ID: 10, Title: "Concert de pagament"},
			20: {ID: 20, Title: "Concert gratuït"},
		},
		searchIDs: []int{10, 20},
	}

	t.Run("explicit none returns zero cards", func(t *testing.T) {
		ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
			functionCall("c1", ToolSearchEvents, `{"category":"concert","municipality":"Barcelona"}`),
			textReply("No he trobat concerts gratuïts aquest cap de setmana.\n[RECOMMENDED_IDS: none]"),
		}}
		s := NewAssistantService(&fakeChatbotRepository{}, cat, ai, func() time.Time { return fixedNow }, ReplyTimeout)
		exchange, err := s.SendMessage(context.Background(), validUserID, "concerts gratuïts a Barcelona", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(exchange.AssistantMessage.RecommendedEvents) != 0 {
			t.Fatalf("expected 0 cards, got %+v", exchange.AssistantMessage.RecommendedEvents)
		}
	})

	t.Run("hallucinated and duplicate IDs are filtered out", func(t *testing.T) {
		ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
			functionCall("c1", ToolSearchEvents, `{"category":"concert","municipality":"Barcelona"}`),
			textReply("Et recomano **Concert gratuït**.\n[RECOMMENDED_IDS: 9999, 20, 20, 8888]"),
		}}
		s := NewAssistantService(&fakeChatbotRepository{}, cat, ai, func() time.Time { return fixedNow }, ReplyTimeout)
		exchange, err := s.SendMessage(context.Background(), validUserID, "concerts gratuïts a Barcelona", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := eventIDs(exchange.AssistantMessage.RecommendedEvents); !slices.Equal(got, []int{20}) {
			t.Fatalf("recommended IDs = %v, want [20]", got)
		}
	})
}

func TestSendMessage_FallbackMatchesTitlesInTextOrderWhenTagMissing(t *testing.T) {
	cat := &titledFakeCatalogue{
		byID: map[int]event_model.Event{
			10: {ID: 10, Title: "Espai Vapor"},
			20: {ID: 20, Title: "Les parets criden"},
			30: {ID: 30, Title: "La invenció del temps"},
			40: {ID: 40, Title: "Exposició MACBA"},
			50: {ID: 50, Title: "Clave Gaudí"},
		},
		searchIDs: []int{10, 20, 30, 40, 50},
	}
	// Model forgot the [RECOMMENDED_IDS: ...] trailer, but explicitly named 3 events in its text.
	ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
		functionCall("c1", ToolSearchEvents, `{"category":"exposició","municipality":"Barcelona"}`),
		textReply("Et recomano **Clave Gaudí**, **Les parets criden** i **Exposició MACBA**."),
	}}

	s := NewAssistantService(&fakeChatbotRepository{}, cat, ai, func() time.Time { return fixedNow }, ReplyTimeout)
	exchange, err := s.SendMessage(context.Background(), validUserID, "exposicions a Barcelona", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := eventIDs(exchange.AssistantMessage.RecommendedEvents); !slices.Equal(got, []int{50, 20, 40}) {
		t.Fatalf("recommended IDs = %v, want [50 20 40] matching exact text order", got)
	}
}

func TestSendMessage_ExcludesAlreadyShownEventsAcrossTurns(t *testing.T) {
	startedAt := fixedNow.Add(-10 * time.Second)
	allEvents := map[int]event_model.Event{
		1: {ID: 1, Title: "Exposició 1"},
		2: {ID: 2, Title: "Exposició 2"},
		3: {ID: 3, Title: "Exposició 3"},
		4: {ID: 4, Title: "Exposició 4"},
		5: {ID: 5, Title: "Exposició 5"},
		6: {ID: 6, Title: "Exposició 6"},
	}

	// Turn 2: events 1, 2, 3 were already shown in history.
	repo := &fakeChatbotRepository{
		startedAt: startedAt,
		history: []chatbot_model.Message{
			{Sender: chatbot_model.SenderUser, Text: "exposicions a Barcelona"},
			{
				Sender:            chatbot_model.SenderAssistant,
				Text:              "Aquí tens 3 exposicions.",
				RecommendedEvents: []event_model.Event{allEvents[1], allEvents[2], allEvents[3]},
			},
		},
	}
	cat := &titledFakeCatalogue{
		byID:      allEvents,
		searchIDs: []int{1, 2, 3, 4, 5, 6},
	}
	ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
		functionCall("c1", ToolSearchEvents, `{"category":"exposició","municipality":"Barcelona"}`),
		textReply("Cap problema! Aquí tens 3 opcions noves:\n- **Exposició 4**\n- **Exposició 5**\n- **Exposició 6**\n[RECOMMENDED_IDS: 4, 5, 6]"),
	}}

	s := NewAssistantService(repo, cat, ai, func() time.Time { return fixedNow }, ReplyTimeout)
	exchange, err := s.SendMessage(context.Background(), validUserID, "no m'agraden, ensenya-me'n més", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify Search requested extra rows to compensate for the 3 already-shown IDs.
	if len(cat.searches) != 1 || cat.searches[0].Limit != defaultLimit+3 {
		t.Fatalf("search filter limit = %+v, want %d", cat.searches, defaultLimit+3)
	}

	// Verify tool result sent to model excluded 1, 2, 3 and only exposed 4, 5, 6.
	toolMsg := ai.requests[1].Messages[len(ai.requests[1].Messages)-1]
	var toolPayload struct {
		Events              []EventSummary `json:"events"`
		AlreadyShownSkipped int            `json:"already-shown-skipped"`
	}
	if err := json.Unmarshal([]byte(toolMsg.Content), &toolPayload); err != nil {
		t.Fatalf("unmarshal tool payload: %v", err)
	}
	if toolPayload.AlreadyShownSkipped != 3 {
		t.Errorf("already-shown-skipped = %d, want 3", toolPayload.AlreadyShownSkipped)
	}
	gotToolIDs := make([]int, len(toolPayload.Events))
	for i, e := range toolPayload.Events {
		gotToolIDs[i] = e.ID
	}
	if !slices.Equal(gotToolIDs, []int{4, 5, 6}) {
		t.Errorf("tool returned IDs = %v, want [4 5 6]", gotToolIDs)
	}

	// Verify final cards are the 3 brand-new events.
	if got := eventIDs(exchange.AssistantMessage.RecommendedEvents); !slices.Equal(got, []int{4, 5, 6}) {
		t.Fatalf("recommended IDs = %v, want [4 5 6]", got)
	}

	// Verify agent history includes [Shown events: ...] while topic classifier does not.
	agentHistoryAssistant := ai.requests[0].Messages[2].Content
	if !strings.Contains(agentHistoryAssistant, "[Shown events: #1 Exposició 1, #2 Exposició 2, #3 Exposició 3]") {
		t.Errorf("agent history missing shown events summary: %q", agentHistoryAssistant)
	}
	topicHistoryAssistant := ai.topicRequests[0].Messages[2].Content
	if strings.Contains(topicHistoryAssistant, "[Shown events:") {
		t.Errorf("topic classifier history should stay plain: %q", topicHistoryAssistant)
	}

	// Turn 3: all 6 events are now in history -> search returns 0 unseen and reports 6 skipped.
	repo.history = append(repo.history, exchange.UserMessage, exchange.AssistantMessage)
	ai3 := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
		functionCall("c2", ToolSearchEvents, `{"category":"exposició","municipality":"Barcelona"}`),
		textReply("Ja t'he ensenyat totes les exposicions disponibles a Barcelona.\n[RECOMMENDED_IDS: none]"),
	}}
	cat3 := &titledFakeCatalogue{byID: allEvents, searchIDs: []int{1, 2, 3, 4, 5, 6}}
	s3 := NewAssistantService(repo, cat3, ai3, func() time.Time { return fixedNow }, ReplyTimeout)
	ex3, err := s3.SendMessage(context.Background(), validUserID, "més opcions", false)
	if err != nil {
		t.Fatalf("turn 3 error: %v", err)
	}
	if len(ex3.AssistantMessage.RecommendedEvents) != 0 {
		t.Fatalf("turn 3 must not repeat already shown events: %+v", ex3.AssistantMessage.RecommendedEvents)
	}
	toolMsg3 := ai3.requests[1].Messages[len(ai3.requests[1].Messages)-1]
	if !strings.Contains(toolMsg3.Content, `"already-shown-skipped":6`) {
		t.Errorf("turn 3 tool payload = %s", toolMsg3.Content)
	}
}

func TestParseRecommendedIDsTruncatedAndEchoes(t *testing.T) {
	text, ids, ok := parseRecommendedIDs("Hola\n[RECOMMENDED_IDS: 12, 34")
	if !ok || text != "Hola" || !slices.Equal(ids, []int{12, 34}) {
		t.Fatalf("truncated tag: %q %v %v", text, ids, ok)
	}
	text, _, _ = parseRecommendedIDs("**Clave Gaudí**[RECOMMENDED_IDS: 5]")
	if text != "**Clave Gaudí**" {
		t.Fatalf("bold closing eaten: %q", text)
	}
	text, _, _ = parseRecommendedIDs("Text\n[Shown events: #1 A, #2 B]\n[RECOMMENDED_IDS: none]")
	if text != "Text" {
		t.Fatalf("shown-events echo kept: %q", text)
	}
}

func TestNeedsClarification(t *testing.T) {
	asst := func(text string, n int) chatbot_model.Message {
		m := chatbot_model.Message{Sender: chatbot_model.SenderAssistant, Text: text}
		m.RecommendedEvents = make([]event_model.Event, n)
		return m
	}
	vague := `{"from":"2026-11-28","to":"2026-11-29"}`
	cases := []struct {
		name    string
		history []chatbot_model.Message
		args    string
		want    bool
	}{
		{"fresh vague", nil, vague, true},
		{"has municipality", nil, `{"municipality":"Girona"}`, false},
		{"answering a question", []chatbot_model.Message{asst("Quin tipus de pla?", 0)}, vague, false},
		{"more of earlier recs", []chatbot_model.Message{asst("Aquí tens", 3)}, vague, false},
		{"new vague request after plain answer", []chatbot_model.Message{asst("Hola!", 0)}, vague, true},
	}
	for _, c := range cases {
		if got := needsClarification(c.history, c.args); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}
