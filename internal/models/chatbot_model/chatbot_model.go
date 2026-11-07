package chatbot_model

import (
	"errors"
	"time"

	"api-aprop/internal/models/event_model"
)

// Sender is the author of a chatbot message (MissatgeXatbot.emissor).
type Sender string

const (
	SenderUser      Sender = "user"
	SenderAssistant Sender = "assistant"
)

var (
	ErrInvalidSender           = errors.New("invalid sender")
	ErrOnlyAssistantInterprets = errors.New("R12: only chatbot messages can interpret the query")
	ErrOnlyAssistantRecommends = errors.New("R12: only chatbot messages can recommend events")
	ErrOnlyUserByVoice         = errors.New("R12: only user messages can be sent by voice")
)

// Interpretation holds the search criteria the chatbot understood from the user query.
// Empty strings and nil dates mean the criterion was not given.
type Interpretation struct {
	Category     string
	Municipality string
	DateFrom     *time.Time
	DateTo       *time.Time
}

// IsEmpty reports whether no criterion was interpreted.
func (i Interpretation) IsEmpty() bool {
	return i.Category == "" && i.Municipality == "" && i.DateFrom == nil && i.DateTo == nil
}

// Message is a message of a chatbot conversation (MissatgeXatbot).
type Message struct {
	SentAt            time.Time
	Sender            Sender
	Text              string
	ByVoice           bool
	Interpretation    *Interpretation     // only chatbot messages
	RecommendedEvents []event_model.Event // only chatbot messages
	// Action is the app screen the reply suggests opening (see assistant_service.ScreenFor).
	// Only chatbot messages; it is not stored, so it is empty in a loaded conversation.
	Action string
}

// Validate checks the domain restriction R12.
func (m Message) Validate() error {
	switch m.Sender {
	case SenderUser:
		if m.Interpretation != nil {
			return ErrOnlyAssistantInterprets
		}
		if len(m.RecommendedEvents) > 0 {
			return ErrOnlyAssistantRecommends
		}
		if m.Action != "" {
			return ErrOnlyAssistantRecommends
		}
	case SenderAssistant:
		if m.ByVoice {
			return ErrOnlyUserByVoice
		}
	default:
		return ErrInvalidSender
	}
	return nil
}

// Conversation is a chatbot conversation of a user (ConversaXatbot).
type Conversation struct {
	StartedAt time.Time
	Messages  []Message
}

// Exchange is a user message and the chatbot reply to it.
type Exchange struct {
	ConversationStartedAt time.Time
	UserMessage           Message
	AssistantMessage      Message
}
