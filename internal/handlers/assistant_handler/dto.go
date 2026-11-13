package assistant_handler

import (
	"time"

	"api-aprop/internal/models/chatbot_model"
	"api-aprop/internal/models/event_model"
)

type sendMessageRequest struct {
	Text    string `json:"text"`
	ByVoice bool   `json:"by-voice"`
}

type feedbackRequest struct {
	ConversationStartedAt time.Time `json:"conversation-started-at"`
	SentAt                time.Time `json:"sent-at"` // of the assistant message that is rated
	Helpful               *bool     `json:"helpful"`
}

// actionResponse is a screen of the app that the reply suggests opening.
type actionResponse struct {
	Screen string `json:"screen"`
}

type interpretationResponse struct {
	Category     string `json:"category,omitempty"`
	Municipality string `json:"municipality,omitempty"`
	DateFrom     string `json:"date-from,omitempty"`
	DateTo       string `json:"date-to,omitempty"`
	DateRange    string `json:"date-range,omitempty"` // label in Catalan, e.g. "del 24/10/2026 al 25/10/2026"
}

type messageResponse struct {
	SentAt            time.Time               `json:"sent-at"`
	Sender            chatbot_model.Sender    `json:"sender"`
	Text              string                  `json:"text"`
	ByVoice           bool                    `json:"by-voice"`
	Interpretation    *interpretationResponse `json:"interpretation,omitempty"`
	RecommendedEvents []event_model.Event     `json:"recommended-events,omitempty"`
	Action            *actionResponse         `json:"action,omitempty"` // only in a reply just sent, not in a loaded conversation
}

type exchangeResponse struct {
	ConversationStartedAt time.Time       `json:"conversation-started-at"`
	UserMessage           messageResponse `json:"user-message"`
	AssistantMessage      messageResponse `json:"assistant-message"`
}

type conversationResponse struct {
	StartedAt time.Time         `json:"started-at"`
	Messages  []messageResponse `json:"messages"`
}

const (
	dateLayout      = "2006-01-02"
	labelDateLayout = "2/1/2006"
)

func toExchangeResponse(e chatbot_model.Exchange) exchangeResponse {
	return exchangeResponse{
		ConversationStartedAt: e.ConversationStartedAt,
		UserMessage:           toMessageResponse(e.UserMessage),
		AssistantMessage:      toMessageResponse(e.AssistantMessage),
	}
}

func toConversationResponse(c chatbot_model.Conversation) conversationResponse {
	messages := make([]messageResponse, 0, len(c.Messages))
	for _, m := range c.Messages {
		messages = append(messages, toMessageResponse(m))
	}
	return conversationResponse{StartedAt: c.StartedAt, Messages: messages}
}

func toMessageResponse(m chatbot_model.Message) messageResponse {
	r := messageResponse{
		SentAt:            m.SentAt,
		Sender:            m.Sender,
		Text:              m.Text,
		ByVoice:           m.ByVoice,
		RecommendedEvents: m.RecommendedEvents,
	}
	if m.Action != "" {
		r.Action = &actionResponse{Screen: m.Action}
	}
	if i := m.Interpretation; i != nil && (*i != chatbot_model.Interpretation{}) {
		r.Interpretation = &interpretationResponse{
			Category:     i.Category,
			Municipality: i.Municipality,
			DateFrom:     formatDate(i.DateFrom),
			DateTo:       formatDate(i.DateTo),
			DateRange:    dateRangeLabel(i.DateFrom, i.DateTo),
		}
	}
	return r
}

func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(dateLayout)
}

func dateRangeLabel(from, to *time.Time) string {
	switch {
	case from != nil && to != nil && from.Equal(*to):
		return "el " + from.Format(labelDateLayout)
	case from != nil && to != nil:
		return withPreposition("de", *from) + " " + withPreposition("a", *to)
	case from != nil:
		return "a partir " + withPreposition("de", *from)
	case to != nil:
		return "fins " + withPreposition("a", *to)
	default:
		return ""
	}
}

// withPreposition contracts "de"/"a" with the article: "del 24/10", but "de l'1/11" and "a l'11/11".
func withPreposition(preposition string, t time.Time) string {
	date := t.Format(labelDateLayout)
	if t.Day() == 1 || t.Day() == 11 {
		return preposition + " l'" + date
	}
	if preposition == "de" {
		return "del " + date
	}
	return "al " + date
}
