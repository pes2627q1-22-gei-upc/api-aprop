CREATE TABLE "public"."userstate" (
  "userid" uuid    NOT NULL,
  "state"  integer,
  CONSTRAINT "userstate_pkey" PRIMARY KEY (userid),
  CONSTRAINT "userstate_state_fkey" FOREIGN KEY (state) REFERENCES public.accountstates(id)
);

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."userstate" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."userstate" TO "service_role";

REVOKE ALL ON TABLE "public"."userstate" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."userstate" TO "postgres";
