CREATE TABLE "public"."levels" (
  "id"                 integer           GENERATED ALWAYS AS IDENTITY NOT NULL,
  "name"               character varying NOT NULL,
  "experiencerequired" integer           NOT NULL,
  CONSTRAINT "levels_experiencerequired_key" UNIQUE (experiencerequired),
  CONSTRAINT "levels_name_key" UNIQUE (name),
  CONSTRAINT "levels_pkey" PRIMARY KEY (id)
);

ALTER TABLE "public"."levels"
  ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON SEQUENCE "public"."levels_id_seq" FROM "anon";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."levels_id_seq" TO "anon";

REVOKE ALL ON SEQUENCE "public"."levels_id_seq" FROM "authenticated";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."levels_id_seq" TO "authenticated";

REVOKE ALL ON SEQUENCE "public"."levels_id_seq" FROM "postgres";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."levels_id_seq" TO "postgres";

REVOKE ALL ON SEQUENCE "public"."levels_id_seq" FROM "service_role";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."levels_id_seq" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."levels" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."levels" TO "service_role";

REVOKE ALL ON TABLE "public"."levels" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."levels" TO "postgres";
