CREATE TABLE "public"."chatbotconversations" (
  "userid"    uuid                     NOT NULL,
  "startedat" timestamp with time zone NOT NULL,
  CONSTRAINT "chatbotconversations_pkey" PRIMARY KEY (userid, startedat)
);

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."chatbotconversations" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."chatbotconversations" TO "service_role";

REVOKE ALL ON TABLE "public"."chatbotconversations" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."chatbotconversations" TO "postgres";
