CREATE TABLE "public"."settings" (
  "id" integer GENERATED ALWAYS AS IDENTITY NOT NULL,
  CONSTRAINT "settings_pkey" PRIMARY KEY (id)
);

REVOKE ALL ON SEQUENCE "public"."settings_id_seq" FROM "anon";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."settings_id_seq" TO "anon";

REVOKE ALL ON SEQUENCE "public"."settings_id_seq" FROM "authenticated";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."settings_id_seq" TO "authenticated";

REVOKE ALL ON SEQUENCE "public"."settings_id_seq" FROM "postgres";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."settings_id_seq" TO "postgres";

REVOKE ALL ON SEQUENCE "public"."settings_id_seq" FROM "service_role";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."settings_id_seq" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."settings" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."settings" TO "service_role";

REVOKE ALL ON TABLE "public"."settings" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."settings" TO "postgres";
