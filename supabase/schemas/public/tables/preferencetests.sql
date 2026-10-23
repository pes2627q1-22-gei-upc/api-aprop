CREATE TABLE "public"."preferencetests" (
  "userid"    uuid                        NOT NULL,
  "datetaken" timestamp without time zone NOT NULL,
  CONSTRAINT "preferencetests_pkey" PRIMARY KEY (userid, datetaken)
);

ALTER TABLE "public"."preferencetests"
  ENABLE ROW LEVEL SECURITY;

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."preferencetests" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."preferencetests" TO "service_role";

REVOKE ALL ON TABLE "public"."preferencetests" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."preferencetests" TO "postgres";
