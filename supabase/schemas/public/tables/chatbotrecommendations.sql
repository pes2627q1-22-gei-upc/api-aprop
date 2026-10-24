CREATE TABLE "public"."chatbotrecommendations" (
  "userid"                uuid                     NOT NULL,
  "conversationstartedat" timestamp with time zone NOT NULL,
  "sentat"                timestamp with time zone NOT NULL,
  "sender"                character varying        NOT NULL DEFAULT 'assistant'::character varying,
  "event"                 integer                  NOT NULL,
  "position"              integer                  NOT NULL,
  CONSTRAINT "chatbotrecommendations_pkey" PRIMARY KEY (userid, conversationstartedat, sentat, EVENT),
  CONSTRAINT "chatbotrecommendations_sender_check" CHECK (((sender)::text = 'assistant'::text)),
  CONSTRAINT "chatbotrecommendations_userid_conversationstartedat_sentat_fkey" FOREIGN KEY (userid, conversationstartedat, sentat, sender)
    REFERENCES public.chatbotmessages(userid, conversationstartedat, sentat, sender) ON DELETE CASCADE,
  CONSTRAINT "chatbotrecommendations_event_fkey" FOREIGN KEY (EVENT) REFERENCES public.events(id)
);

ALTER TABLE "public"."chatbotrecommendations"
  ENABLE ROW LEVEL SECURITY;

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."chatbotrecommendations" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."chatbotrecommendations" TO "service_role";

REVOKE ALL ON TABLE "public"."chatbotrecommendations" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."chatbotrecommendations" TO "postgres";
