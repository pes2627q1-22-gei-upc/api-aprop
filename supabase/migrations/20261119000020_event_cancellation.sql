-- Cancel·lació of an unofficial event by its creator (Taiga #180, DP-041). Cancelled events cannot be modified.
ALTER TYPE UnofficialEventState ADD VALUE IF NOT EXISTS 'Cancelled';
-- NotificacióEsdeveniment.tipus Cancel·lació. It is inserted here because NotificationRepository rebuilds the types when
-- their number is not NumNotificationTypes, which fails once notifications reference them.
INSERT INTO NotificationTypes (Name) VALUES ('Cancelacio') ON CONFLICT (Name) DO NOTHING;
