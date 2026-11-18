-- State of an unofficial event (NoOficial.estat). Only Active events can be modified; R7 decides when one is Withdrawn
DO $$ BEGIN
    CREATE TYPE UnofficialEventState AS ENUM ('Active', 'Withdrawn');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
ALTER TABLE UnofficialEvents ADD COLUMN IF NOT EXISTS State UnofficialEventState NOT NULL DEFAULT 'Active';
