CREATE TABLE "public"."eventratings" (
  "userid"     uuid                        NOT NULL,
  "event"      integer                     NOT NULL,
  "ratingdate" timestamp without time zone NOT NULL,
  "rating"     integer                     NOT NULL,
  CONSTRAINT "eventratings_pkey" PRIMARY KEY (userid, EVENT),
  CONSTRAINT "eventratings_rating_check" CHECK (((rating >= 0) AND (rating <= 5))),
  CONSTRAINT "eventratings_event_fkey" FOREIGN KEY (EVENT) REFERENCES public.events(id)
);

ALTER TABLE "public"."eventratings"
  ENABLE ROW LEVEL SECURITY;

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."eventratings" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."eventratings" TO "service_role";

REVOKE ALL ON TABLE "public"."eventratings" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."eventratings" TO "postgres";
