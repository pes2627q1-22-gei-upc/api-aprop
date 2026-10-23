CREATE TABLE "public"."notifications" (
  "userid"           uuid                        NOT NULL,
  "notificationdate" timestamp without time zone NOT NULL,
  "friend"           uuid,
  "requestdate"      timestamp without time zone,
  "event"            integer,
  "type"             integer,
  CONSTRAINT "notifications_check" CHECK ((userid <> friend)),
  CONSTRAINT "notifications_event_fkey" FOREIGN KEY (EVENT) REFERENCES public.events(id),
  CONSTRAINT "notifications_friend_userid_requestdate_fkey" FOREIGN KEY (friend, userid, requestdate) REFERENCES public.friendrequests(sender, receiver, datesent),
  CONSTRAINT "notifications_pkey" PRIMARY KEY (userid, notificationdate),
  CONSTRAINT "notifications_type_fkey" FOREIGN KEY (TYPE) REFERENCES public.notificationtypes(id)
);

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."notifications" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."notifications" TO "service_role";

REVOKE ALL ON TABLE "public"."notifications" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."notifications" TO "postgres";
