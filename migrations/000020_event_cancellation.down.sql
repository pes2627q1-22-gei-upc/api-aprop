DELETE FROM Notifications WHERE Type = (SELECT ID FROM NotificationTypes WHERE Name = 'Cancelacio');
DELETE FROM NotificationTypes WHERE Name = 'Cancelacio';
-- Postgres cannot drop an enum value: the value stays, but no event uses it.
UPDATE UnofficialEvents SET State = 'Active' WHERE State::text = 'Cancelled';
