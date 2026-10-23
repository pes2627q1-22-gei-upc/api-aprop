CREATE TABLE "public"."preferencetestchoices" (
  "testuser"  uuid                        NOT NULL,
  "datetaken" timestamp without time zone NOT NULL,
  "event"     integer                     NOT NULL,
  "likes"     boolean                     NOT NULL,
  CONSTRAINT "preferencetestchoices_event_fkey" FOREIGN KEY (EVENT) REFERENCES public.events(id),
  CONSTRAINT "preferencetestchoices_pkey" PRIMARY KEY (testuser, datetaken, EVENT),
  CONSTRAINT "preferencetestchoices_testuser_datetaken_fkey" FOREIGN KEY (testuser, datetaken) REFERENCES public.preferencetests(userid, datetaken)
);

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."preferencetestchoices" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."preferencetestchoices" TO "service_role";

REVOKE ALL ON TABLE "public"."preferencetestchoices" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."preferencetestchoices" TO "postgres";
