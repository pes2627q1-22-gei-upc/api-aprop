CREATE TABLE "public"."accountstates" (
  "id"   integer           GENERATED ALWAYS AS IDENTITY NOT NULL,
  "name" character varying NOT NULL,
  CONSTRAINT "accountstates_name_key" UNIQUE (name),
  CONSTRAINT "accountstates_pkey" PRIMARY KEY (id)
);

REVOKE ALL ON SEQUENCE "public"."accountstates_id_seq" FROM "anon";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."accountstates_id_seq" TO "anon";

REVOKE ALL ON SEQUENCE "public"."accountstates_id_seq" FROM "authenticated";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."accountstates_id_seq" TO "authenticated";

REVOKE ALL ON SEQUENCE "public"."accountstates_id_seq" FROM "postgres";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."accountstates_id_seq" TO "postgres";

REVOKE ALL ON SEQUENCE "public"."accountstates_id_seq" FROM "service_role";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."accountstates_id_seq" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."accountstates" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."accountstates" TO "service_role";

REVOKE ALL ON TABLE "public"."accountstates" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."accountstates" TO "postgres";
