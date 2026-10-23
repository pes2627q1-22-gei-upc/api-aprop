CREATE TABLE "public"."signup" (
  "userid"     uuid                        NOT NULL,
  "event"      integer                     NOT NULL,
  "signupdate" timestamp without time zone NOT NULL,
  CONSTRAINT "signup_event_fkey" FOREIGN KEY (EVENT) REFERENCES public.events(id),
  CONSTRAINT "signup_pkey" PRIMARY KEY (userid, EVENT, signupdate)
);

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."signup" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."signup" TO "service_role";

REVOKE ALL ON TABLE "public"."signup" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."signup" TO "postgres";
