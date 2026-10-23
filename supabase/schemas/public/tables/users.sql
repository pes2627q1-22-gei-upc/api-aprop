CREATE TABLE "public"."users" (
  "id"             integer                     GENERATED ALWAYS AS IDENTITY NOT NULL,
  "firebaseuid"    character varying           NOT NULL,
  "username"       character varying           NOT NULL,
  "email"          character varying,
  "profilepicture" character varying,
  "biography"      character varying,
  "dateregistered" timestamp without time zone NOT NULL DEFAULT now(),
  "accountstate"   integer                     NOT NULL,
  "experience"     integer                     NOT NULL DEFAULT 0,
  "level"          integer,
  "rating"         real,
  "settings"       integer                     NOT NULL,
  CONSTRAINT "users_accountstate_fkey" FOREIGN KEY (accountstate) REFERENCES public.accountstates(id),
  CONSTRAINT "users_firebaseuid_key" UNIQUE (firebaseuid),
  CONSTRAINT "users_level_fkey" FOREIGN KEY (level) REFERENCES public.levels(id),
  CONSTRAINT "users_pkey" PRIMARY KEY (id),
  CONSTRAINT "users_settings_fkey" FOREIGN KEY (settings) REFERENCES public.settings(id),
  CONSTRAINT "users_settings_key" UNIQUE (settings),
  CONSTRAINT "users_username_key" UNIQUE (username)
);

REVOKE ALL ON SEQUENCE "public"."users_id_seq" FROM "anon";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."users_id_seq" TO "anon";

REVOKE ALL ON SEQUENCE "public"."users_id_seq" FROM "authenticated";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."users_id_seq" TO "authenticated";

REVOKE ALL ON SEQUENCE "public"."users_id_seq" FROM "postgres";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."users_id_seq" TO "postgres";

REVOKE ALL ON SEQUENCE "public"."users_id_seq" FROM "service_role";

GRANT SELECT, UPDATE, USAGE ON SEQUENCE "public"."users_id_seq" TO "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."users" TO "anon", "authenticated";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."users" TO "service_role";

REVOKE ALL ON TABLE "public"."users" FROM "postgres";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."users" TO "postgres";
