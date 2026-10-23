CREATE TABLE "public"."friendrequests" (
  "sender"   uuid                        NOT NULL,
  "receiver" uuid                        NOT NULL,
  "datesent" timestamp without time zone NOT NULL,
  CONSTRAINT "friendrequests_check" CHECK ((sender <> receiver)),
  CONSTRAINT "friendrequests_pkey" PRIMARY KEY (sender, receiver, datesent)
);

ALTER TABLE "public"."friendrequests"
  ENABLE ROW LEVEL SECURITY;

ALTER TABLE "public"."friendrequests"
  ADD COLUMN "state" public.friendrequeststate NOT NULL DEFAULT 'Pending'::public.friendrequeststate;

ALTER TABLE "public"."friendrequests"
  ADD COLUMN "dateresolved" timestamp WITHOUT time zone;

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."friendrequests" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."friendrequests" TO "service_role";

REVOKE ALL ON TABLE "public"."friendrequests" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."friendrequests" TO "postgres";
