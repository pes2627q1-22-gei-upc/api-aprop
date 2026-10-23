CREATE TABLE "public"."userpreferences" (
  "userid"   uuid    NOT NULL,
  "category" integer NOT NULL,
  "weight"   real    NOT NULL DEFAULT 1,
  CONSTRAINT "userpreferences_category_fkey" FOREIGN KEY (category) REFERENCES public.categories(id),
  CONSTRAINT "userpreferences_pkey" PRIMARY KEY (userid, category)
);

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."userpreferences" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."userpreferences" TO "service_role";

REVOKE ALL ON TABLE "public"."userpreferences" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."userpreferences" TO "postgres";
