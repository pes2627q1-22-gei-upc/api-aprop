CREATE TABLE "public"."userratings" (
  "rater"      uuid                        NOT NULL,
  "rated"      uuid                        NOT NULL,
  "event"      integer                     NOT NULL,
  "ratingdate" timestamp without time zone NOT NULL,
  CONSTRAINT "userratings_check" CHECK ((rater <> rated)),
  CONSTRAINT "userratings_event_fkey" FOREIGN KEY (EVENT) REFERENCES public.events(id),
  CONSTRAINT "userratings_pkey" PRIMARY KEY (rater, rated, EVENT)
);

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."userratings" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."userratings" TO "service_role";

REVOKE ALL ON TABLE "public"."userratings" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."userratings" TO "postgres";
