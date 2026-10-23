CREATE TABLE "public"."friendships" (
  "user1" uuid NOT NULL,
  "user2" uuid NOT NULL,
  CONSTRAINT "friendships_check" CHECK ((user1 <> user2)),
  CONSTRAINT "friendships_pkey" PRIMARY KEY (user1, user2)
);

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."friendships" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."friendships" TO "service_role";

REVOKE ALL ON TABLE "public"."friendships" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."friendships" TO "postgres";
