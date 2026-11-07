-- User rating (thumbs up/down) of a chatbot reply. One rating per reply; rating again replaces it.
CREATE TABLE ChatbotFeedback (
    UserID UUID NOT NULL,
    ConversationStartedAt Timestamptz NOT NULL,
    SentAt Timestamptz NOT NULL,
    Sender Varchar NOT NULL DEFAULT 'assistant',
    Helpful Boolean NOT NULL,
    CreatedAt Timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (UserID, ConversationStartedAt, SentAt),
    FOREIGN KEY (UserID, ConversationStartedAt, SentAt, Sender)
        REFERENCES ChatbotMessages(UserID, ConversationStartedAt, SentAt, Sender) ON DELETE CASCADE,
    -- only chatbot replies can be rated
    CHECK (Sender = 'assistant')
);

ALTER TABLE public.chatbotfeedback ENABLE ROW LEVEL SECURITY;
