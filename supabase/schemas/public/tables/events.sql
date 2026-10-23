CREATE TABLE "public"."events" (
  "id"          integer           GENERATED ALWAYS AS IDENTITY NOT NULL,
  "title"       character varying NOT NULL,
  "subtitle"    character varying,
  "description" character varying,
  "schedule"    character varying,
  "url"         character varying,
  "tickets"     character varying,
  "ticketsurl"  character varying,
  "free"        boolean,
  "permanent"   boolean,
  "relatedurls" character varying,
  "urlnames"    character varying,
  "images"      character varying,
  "videos"      character varying,
  "email"       character varying,
  "phone"       character varying,
  "activityurl" character varying,
  "startdate"   date,
  "enddate"     date,
  CONSTRAINT "events_check" CHECK ((enddate >= startdate)),
  CONSTRAINT "events_pkey" PRIMARY KEY (id)
);

ALTER TABLE "public"."events"
  ADD COLUMN "modality" public.modalitytype;

ALTER TABLE "public"."events"
  ADD COLUMN "place" character varying;

ALTER TABLE "public"."events"
  ADD COLUMN "latitude" real;

ALTER TABLE "public"."events"
  ADD COLUMN "longitude" real;

ALTER TABLE "public"."events"
  ADD COLUMN "postalcode" character varying;

ALTER TABLE "public"."events"
  ADD COLUMN "municipality" character varying;

ALTER TABLE "public"."events"
  ADD COLUMN "region" character varying;

ALTER TABLE "public"."events"
  ADD COLUMN "address" character varying;

ALTER TABLE "public"."events"
  ADD CONSTRAINT "valid_lat" CHECK (((latitude <= (90)::double precision) AND (latitude >= ('-90'::integer)::double precision)));

ALTER TABLE "public"."events"
  ADD CONSTRAINT "valid_long" CHECK (((longitude <= (180)::double precision) AND (longitude >= ('-180'::integer)::double precision)));

REVOKE ALL ON SEQUENCE "public"."events_id_seq" FROM "anon";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."events_id_seq" TO "anon";

REVOKE ALL ON SEQUENCE "public"."events_id_seq" FROM "authenticated";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."events_id_seq" TO "authenticated";

REVOKE ALL ON SEQUENCE "public"."events_id_seq" FROM "postgres";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."events_id_seq" TO "postgres";

REVOKE ALL ON SEQUENCE "public"."events_id_seq" FROM "service_role";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."events_id_seq" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."events" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."events" TO "service_role";

REVOKE ALL ON TABLE "public"."events" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."events" TO "postgres";
