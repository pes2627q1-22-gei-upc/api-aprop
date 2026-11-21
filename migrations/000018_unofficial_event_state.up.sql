-- State of an unofficial event (NoOficial.estat). Only Active events can be modified; R7 decides when one is Withdrawn

    CREATE TYPE UnofficialEventState AS ENUM ('Active', 'Withdrawn');

ALTER TABLE UnofficialEvents ADD COLUMN IF NOT EXISTS State UnofficialEventState NOT NULL DEFAULT 'Active';
