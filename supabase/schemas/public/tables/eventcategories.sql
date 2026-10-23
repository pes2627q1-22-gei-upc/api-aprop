CREATE TABLE "public"."eventcategories" (
  "event"    integer NOT NULL,
  "category" integer NOT NULL,
  CONSTRAINT "eventcategories_category_fkey" FOREIGN KEY (category) REFERENCES public.categories(id),
  CONSTRAINT "eventcategories_pkey" PRIMARY KEY (EVENT, category),
  CONSTRAINT "eventcategories_event_fkey" FOREIGN KEY (EVENT) REFERENCES public.events(id)
);

ALTER TABLE "public"."eventcategories"
  ENABLE ROW LEVEL SECURITY;

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."eventcategories" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."eventcategories" TO "service_role";

REVOKE ALL ON TABLE "public"."eventcategories" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."eventcategories" TO "postgres";
