CREATE TABLE "public"."messages" (
  "userid"   uuid                        NOT NULL,
  "event"    integer                     NOT NULL,
  "datesent" timestamp without time zone NOT NULL,
  "message"  character varying           NOT NULL,
  CONSTRAINT "messages_event_fkey" FOREIGN KEY (EVENT) REFERENCES public.events(id),
  CONSTRAINT "messages_pkey" PRIMARY KEY (userid, EVENT, datesent)
);

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."messages" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."messages" TO "service_role";

REVOKE ALL ON TABLE "public"."messages" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."messages" TO "postgres";
