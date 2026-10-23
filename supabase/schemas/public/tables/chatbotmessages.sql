CREATE TABLE "public"."chatbotmessages" (
  "userid"                uuid                     NOT NULL,
  "conversationstartedat" timestamp with time zone NOT NULL,
  "sentat"                timestamp with time zone NOT NULL,
  "sender"                character varying        NOT NULL,
  "text"                  character varying        NOT NULL,
  "byvoice"               boolean                  NOT NULL DEFAULT false,
  "category"              character varying,
  "municipality"          character varying,
  "datefrom"              date,
  "dateto"                date,
  CONSTRAINT "chatbotmessages_check1" CHECK ((((sender)::text = 'assistant'::text) OR ((category IS NULL) AND (municipality IS NULL) AND (datefrom IS NULL) AND (dateto IS NULL)))),
  CONSTRAINT "chatbotmessages_check2" CHECK (((datefrom IS NULL) OR (dateto IS NULL) OR (dateto >= datefrom))),
  CONSTRAINT "chatbotmessages_check" CHECK (((NOT byvoice) OR ((sender)::text = 'user'::text))),
  CONSTRAINT "chatbotmessages_pkey" PRIMARY KEY (userid, conversationstartedat, sentat),
  CONSTRAINT "chatbotmessages_sender_check" CHECK (((sender)::text = ANY ((ARRAY['user'::character varying, 'assistant'::character varying])::text[]))),
  CONSTRAINT "chatbotmessages_userid_conversationstartedat_fkey" FOREIGN KEY (userid, conversationstartedat) REFERENCES public.chatbotconversations(userid, startedat)
    ON DELETE CASCADE,
  CONSTRAINT "chatbotmessages_userid_conversationstartedat_sentat_sender_key" UNIQUE (userid, conversationstartedat, sentat, sender)
);

ALTER TABLE "public"."chatbotmessages"
  ENABLE ROW LEVEL SECURITY;

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."chatbotmessages" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."chatbotmessages" TO "service_role";

REVOKE ALL ON TABLE "public"."chatbotmessages" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."chatbotmessages" TO "postgres";
