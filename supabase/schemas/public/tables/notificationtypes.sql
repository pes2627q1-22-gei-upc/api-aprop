CREATE TABLE "public"."notificationtypes" (
  "id"   integer           GENERATED ALWAYS AS IDENTITY NOT NULL,
  "name" character varying NOT NULL,
  CONSTRAINT "notificationtypes_name_key" UNIQUE (name),
  CONSTRAINT "notificationtypes_pkey" PRIMARY KEY (id)
);

ALTER TABLE "public"."notificationtypes"
  ENABLE ROW LEVEL SECURITY;

REVOKE ALL ON SEQUENCE "public"."notificationtypes_id_seq" FROM "anon";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."notificationtypes_id_seq" TO "anon";

REVOKE ALL ON SEQUENCE "public"."notificationtypes_id_seq" FROM "authenticated";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."notificationtypes_id_seq" TO "authenticated";

REVOKE ALL ON SEQUENCE "public"."notificationtypes_id_seq" FROM "postgres";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."notificationtypes_id_seq" TO "postgres";

REVOKE ALL ON SEQUENCE "public"."notificationtypes_id_seq" FROM "service_role";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."notificationtypes_id_seq" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."notificationtypes" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."notificationtypes" TO "service_role";

REVOKE ALL ON TABLE "public"."notificationtypes" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."notificationtypes" TO "postgres";
