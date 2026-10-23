CREATE TABLE "public"."unofficialevents" (
  "id"          integer                     NOT NULL,
  "creator"     uuid,
  "datecreated" timestamp without time zone,
  CONSTRAINT "unofficialevents_id_fkey" FOREIGN KEY (id) REFERENCES public.events(id),
  CONSTRAINT "unofficialevents_pkey" PRIMARY KEY (id)
);

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."unofficialevents" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."unofficialevents" TO "service_role";

REVOKE ALL ON TABLE "public"."unofficialevents" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."unofficialevents" TO "postgres";
