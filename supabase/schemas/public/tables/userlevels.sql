CREATE TABLE "public"."userlevels" (
  "userid" uuid    NOT NULL,
  "level"  integer,
  CONSTRAINT "userlevels_level_fkey" FOREIGN KEY (level) REFERENCES public.levels(id),
  CONSTRAINT "userlevels_pkey" PRIMARY KEY (userid)
);

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."userlevels" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."userlevels" TO "service_role";

REVOKE ALL ON TABLE "public"."userlevels" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."userlevels" TO "postgres";
