CREATE TABLE "public"."blocked" (
  "blockinguser" uuid NOT NULL,
  "blockeduser"  uuid NOT NULL,
  CONSTRAINT "blocked_check" CHECK ((blockinguser <> blockeduser)),
  CONSTRAINT "blocked_pkey" PRIMARY KEY (blockinguser, blockeduser)
);

ALTER TABLE "public"."blocked"
  ENABLE ROW LEVEL SECURITY;

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."blocked" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."blocked" TO "service_role";

REVOKE ALL ON TABLE "public"."blocked" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."blocked" TO "postgres";
