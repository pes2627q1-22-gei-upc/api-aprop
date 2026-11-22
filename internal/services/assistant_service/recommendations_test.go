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

func newTitledCatalogue(events ...event_model.Event) *titledFakeCatalogue {
	byID := make(map[int]event_model.Event, len(events))
	ids := make([]int, len(events))
	for i, e := range events {
		byID[e.ID] = e
		ids[i] = e.ID
	}
	return &titledFakeCatalogue{byID: byID, searchIDs: ids}
}

func sampleExhibitions() *titledFakeCatalogue {
	return newTitledCatalogue(
		event_model.Event{ID: 10, Title: "Espai Vapor"},
		event_model.Event{ID: 20, Title: "Les parets criden"},
		event_model.Event{ID: 30, Title: "La invenció del temps"},
		event_model.Event{ID: 40, Title: "Exposició MACBA"},
		event_model.Event{ID: 50, Title: "Clave Gaudí"},
	)
}

func searchTurn(args, reply string) []groq_client.ChatCompletionResponse {
	return []groq_client.ChatCompletionResponse{
		functionCall("c1", ToolSearchEvents, args),
		textReply(reply),
	}
}

func sendAssistantMessage(
	t *testing.T,
	repo *fakeChatbotRepository,
	cat *titledFakeCatalogue,
	ai *fakeGroq,
	prompt string,
) chatbot_model.Exchange {
	t.Helper()
	s := NewAssistantService(repo, cat, ai, func() time.Time { return fixedNow }, ReplyTimeout)
	exchange, err := s.SendMessage(context.Background(), validUserID, prompt, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return exchange
}

func TestParseRecommendedIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantText string
		wantIDs  []int
		wantTag  bool
	}{
		{"standard tag on last line", "Et recomano:\n- **Les parets criden**\n- **Clave Gaudí**\n\n[RECOMMENDED_IDS: 42, 17, 88]", "Et recomano:\n- **Les parets criden**\n- **Clave Gaudí**", []int{42, 17, 88}, true},
		{"explicit none tag", "Quin tipus d'activitat et ve de gust i a quina ciutat?\n[RECOMMENDED_IDS: none]", "Quin tipus d'activitat et ve de gust i a quina ciutat?", nil, true},
		{"empty bracket tag", "No he trobat concerts gratuïts.\n[RECOMMENDED_IDS: ]", "No he trobat concerts gratuïts.", nil, true},
		{"json array and markdown wrapping", "Aquí tens opcions.\n`**[RECOMMENDED_IDS: [10, #20, 30]]**`.", "Aquí tens opcions.", []int{10, 20, 30}, true},
		{"standalone line without brackets", "Aquí tens opcions.\nRECOMMENDED_IDS: 5, 3, 1", "Aquí tens opcions.", []int{5, 3, 1}, true},
		{"no tag at all", "Hola! En què et puc ajudar?", "Hola! En què et puc ajudar?", nil, false},
		{"truncated tag", "Hola\n[RECOMMENDED_IDS: 12, 34", "Hola", []int{12, 34}, true},
		{"preserves adjacent bold closing", "**Clave Gaudí**[RECOMMENDED_IDS: 5]", "**Clave Gaudí**", []int{5}, true},
		{"strips echoed shown-events line", "Text\n[Shown events: #1 A, #2 B]\n[RECOMMENDED_IDS: none]", "Text", nil, true},
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
	ai := &fakeGroq{responses: []groq_client.ChatCompletionResponse{
		textReply("Per recomanar-te millor, quin tipus de pla et ve de gust (música, teatre, exposicions...) i en quina ciutat o data?\n[RECOMMENDED_IDS: none]"),
	}}
	cat := newTitledCatalogue(event_model.Event{ID: 1, Title: "Jazz al parc"})

	reply := sendAssistantMessage(t, &fakeChatbotRepository{}, cat, ai, "recomana'm plans").AssistantMessage
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

func TestSendMessage_RecommendationSelection(t *testing.T) {
	fiveConcerts := func() *titledFakeCatalogue {
		return newTitledCatalogue(
			event_model.Event{ID: 1, Title: "Concert U"},
			event_model.Event{ID: 2, Title: "Concert Dos"},
			event_model.Event{ID: 3, Title: "Concert Tres"},
			event_model.Event{ID: 4, Title: "Concert Quatre"},
			event_model.Event{ID: 5, Title: "Concert Cinc"},
		)
	}
	twoConcerts := func() *titledFakeCatalogue {
		return newTitledCatalogue(
			event_model.Event{ID: 10, Title: "Concert de pagament"},
			event_model.Event{ID: 20, Title: "Concert gratuït"},
		)
	}
	multiSearchCat := func() *titledFakeCatalogue {
		cat := newTitledCatalogue(
			event_model.Event{ID: 11, Title: "Jazz al parc"},
			event_model.Event{ID: 12, Title: "Rock a la plaça"},
			event_model.Event{ID: 21, Title: "Exposició MACBA"},
			event_model.Event{ID: 22, Title: "Clave Gaudí"},
		)
		cat.searchBatches = [][]int{{11, 12}, {21, 22}}
		return cat
	}

	const (
		bcnExpoArgs    = `{"category":"exposició","municipality":"Barcelona"}`
		gironaConcArgs = `{"category":"concert","municipality":"Girona"}`
		bcnConcArgs    = `{"category":"concert","municipality":"Barcelona"}`
	)

	tests := []struct {
		name      string
		catalogue func() *titledFakeCatalogue
		responses []groq_client.ChatCompletionResponse
		prompt    string
		wantIDs   []int
	}{
		{
			name:      "cards match exact chosen IDs and order",
			catalogue: sampleExhibitions,
			responses: searchTurn(bcnExpoArgs, "He entès: exposicions a Barcelona.\n- **Les parets criden**\n- **Exposició MACBA**\n- **Clave Gaudí**\n[RECOMMENDED_IDS: 20, 40, 50]"),
			prompt:    "exposicions a Barcelona",
			wantIDs:   []int{20, 40, 50},
		},
		{
			name:      "explicit tag with 5 IDs capped to 3",
			catalogue: fiveConcerts,
			responses: searchTurn(gironaConcArgs, "Aquí tens concerts.\n[RECOMMENDED_IDS: 5, 3, 1, 4, 2]"),
			prompt:    "concerts a Girona",
			wantIDs:   []int{5, 3, 1},
		},
		{
			name:      "no tag fallback capped to 3",
			catalogue: fiveConcerts,
			responses: searchTurn(gironaConcArgs, "He entès: concerts a Girona."),
			prompt:    "concerts a Girona",
			wantIDs:   []int{1, 2, 3},
		},
		{
			name:      "multiple searches combine candidates and preserve order",
			catalogue: multiSearchCat,
			responses: []groq_client.ChatCompletionResponse{
				functionCall("c1", ToolSearchEvents, bcnConcArgs),
				functionCall("c2", ToolSearchEvents, bcnExpoArgs),
				textReply("Et recomano **Jazz al parc**, **Clave Gaudí** i **Exposició MACBA**.\n[RECOMMENDED_IDS: 11, 22, 21]"),
			},
			prompt:  "concerts o exposicions a Barcelona",
			wantIDs: []int{11, 22, 21},
		},
		{
			name:      "explicit none returns zero cards",
			catalogue: twoConcerts,
			responses: searchTurn(bcnConcArgs, "No he trobat concerts gratuïts aquest cap de setmana.\n[RECOMMENDED_IDS: none]"),
			prompt:    "concerts gratuïts a Barcelona",
			wantIDs:   nil,
		},
		{
			name:      "hallucinated and duplicate IDs are filtered out",
			catalogue: twoConcerts,
			responses: searchTurn(bcnConcArgs, "Et recomano **Concert gratuït**.\n[RECOMMENDED_IDS: 9999, 20, 20, 8888]"),
			prompt:    "concerts gratuïts a Barcelona",
			wantIDs:   []int{20},
		},
		{
			name:      "fallback matches titles in text order when tag missing",
			catalogue: sampleExhibitions,
			responses: searchTurn(bcnExpoArgs, "Et recomano **Clave Gaudí**, **Les parets criden** i **Exposició MACBA**."),
			prompt:    "exposicions a Barcelona",
			wantIDs:   []int{50, 20, 40},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ai := &fakeGroq{responses: tc.responses}
			reply := sendAssistantMessage(t, &fakeChatbotRepository{}, tc.catalogue(), ai, tc.prompt).AssistantMessage
			if strings.Contains(reply.Text, "RECOMMENDED_IDS") {
				t.Errorf("tag must not appear in visible text: %q", reply.Text)
			}
			if got := eventIDs(reply.RecommendedEvents); !slices.Equal(got, tc.wantIDs) {
				t.Fatalf("recommended IDs = %v, want %v", got, tc.wantIDs)
			}
		})
	}
}

func TestSendMessage_ExcludesAlreadyShownEventsAcrossTurns(t *testing.T) {
	startedAt := fixedNow.Add(-10 * time.Second)
	events := []event_model.Event{
		{ID: 1, Title: "Exposició 1"},
		{ID: 2, Title: "Exposició 2"},
		{ID: 3, Title: "Exposició 3"},
		{ID: 4, Title: "Exposició 4"},
		{ID: 5, Title: "Exposició 5"},
		{ID: 6, Title: "Exposició 6"},
	}

	// Turn 2: events 1, 2, 3 were already shown in history.
	repo := &fakeChatbotRepository{
		startedAt: startedAt,
		history: []chatbot_model.Message{
			{Sender: chatbot_model.SenderUser, Text: "exposicions a Barcelona"},
			{
				Sender:            chatbot_model.SenderAssistant,
				Text:              "Aquí tens 3 exposicions.",
				RecommendedEvents: events[:3],
			},
		},
	}
	const args = `{"category":"exposició","municipality":"Barcelona"}`
	cat := newTitledCatalogue(events...)
	ai := &fakeGroq{responses: searchTurn(args, "Cap problema! Aquí tens 3 opcions noves:\n- **Exposició 4**\n- **Exposició 5**\n- **Exposició 6**\n[RECOMMENDED_IDS: 4, 5, 6]")}

	exchange := sendAssistantMessage(t, repo, cat, ai, "no m'agraden, ensenya-me'n més")

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
	ai3 := &fakeGroq{responses: searchTurn(args, "Ja t'he ensenyat totes les exposicions disponibles a Barcelona.\n[RECOMMENDED_IDS: none]")}
	ex3 := sendAssistantMessage(t, repo, newTitledCatalogue(events...), ai3, "més opcions")
	if len(ex3.AssistantMessage.RecommendedEvents) != 0 {
		t.Fatalf("turn 3 must not repeat already shown events: %+v", ex3.AssistantMessage.RecommendedEvents)
	}
	toolMsg3 := ai3.requests[1].Messages[len(ai3.requests[1].Messages)-1]
	if !strings.Contains(toolMsg3.Content, `"already-shown-skipped":6`) {
		t.Errorf("turn 3 tool payload = %s", toolMsg3.Content)
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
