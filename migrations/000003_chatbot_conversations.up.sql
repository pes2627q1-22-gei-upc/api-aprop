-- ConversaXatbot: R1 key (Usuari + dataInici)
CREATE TABLE ChatbotConversations (
    UserID UUID NOT NULL,
    StartedAt Timestamptz NOT NULL,
    PRIMARY KEY (UserID, StartedAt)
);

-- MissatgeXatbot: R1 key (Conversa + dataEnviament)
CREATE TABLE ChatbotMessages (
    UserID UUID NOT NULL,
    ConversationStartedAt Timestamptz NOT NULL,
    SentAt Timestamptz NOT NULL,
    Sender Varchar NOT NULL,
    Text Varchar NOT NULL,
    ByVoice Boolean NOT NULL DEFAULT false,
    Category Varchar,
    Municipality Varchar,
    DateFrom Date,
    DateTo Date,
    PRIMARY KEY (UserID, ConversationStartedAt, SentAt),
    UNIQUE (UserID, ConversationStartedAt, SentAt, Sender),
    FOREIGN KEY (UserID, ConversationStartedAt) REFERENCES ChatbotConversations(UserID, StartedAt) ON DELETE CASCADE,
    CHECK (Sender IN ('user', 'assistant')),
    -- R12: only user messages can be sent by voice
    CHECK (NOT ByVoice OR Sender = 'user'),
    -- Only the chatbot interprets the query
    CHECK (Sender = 'assistant' OR (Category IS NULL AND Municipality IS NULL AND DateFrom IS NULL AND DateTo IS NULL)),
    CHECK (DateFrom IS NULL OR DateTo IS NULL OR DateTo >= DateFrom)
);

-- Events recommended by a chatbot message
CREATE TABLE ChatbotRecommendations (
    UserID UUID NOT NULL,
    ConversationStartedAt Timestamptz NOT NULL,
    SentAt Timestamptz NOT NULL,
    Sender Varchar NOT NULL DEFAULT 'assistant',
    Event Integer NOT NULL REFERENCES Events(ID),
    Position Integer NOT NULL,
    PRIMARY KEY (UserID, ConversationStartedAt, SentAt, Event),
    FOREIGN KEY (UserID, ConversationStartedAt, SentAt, Sender)
        REFERENCES ChatbotMessages(UserID, ConversationStartedAt, SentAt, Sender) ON DELETE CASCADE,
    -- R12: only chatbot messages can recommend events
    CHECK (Sender = 'assistant')
);
