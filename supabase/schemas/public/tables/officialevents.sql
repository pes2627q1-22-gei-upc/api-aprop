CREATE TABLE "public"."officialevents" (
  "id"   integer NOT NULL,
  "code" bigint,
  CONSTRAINT "officialevents_id_fkey" FOREIGN KEY (id) REFERENCES public.events(id),
  CONSTRAINT "officialevents_pkey" PRIMARY KEY (id)
);

ALTER TABLE "public"."officialevents"
  ENABLE ROW LEVEL SECURITY;

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."officialevents" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."officialevents" TO "service_role";

REVOKE ALL ON TABLE "public"."officialevents" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."officialevents" TO "postgres";
