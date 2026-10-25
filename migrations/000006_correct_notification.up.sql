DO $$
DECLARE
    default_type_id INTEGER;
BEGIN
    SELECT id
    INTO default_type_id
    FROM notificationtypes
    ORDER BY id
    LIMIT 1;

    IF default_type_id IS NOT NULL THEN
        UPDATE notifications
        SET type = default_type_id
        WHERE type IS NULL;
    ELSE
        DELETE FROM notifications;
    END IF;

    ALTER TABLE notifications
        ALTER COLUMN type SET NOT NULL;
END $$;

delete from notifications;
alter table notifications drop constraint notifications_pkey;
alter table notifications add column id integer generated always as identity primary key;
alter table notifications add column read boolean not null default false;
alter table notifications drop column userid;
alter table notifications add column userid integer not null references users(id);
alter table notifications drop column friend;
alter table notifications add column friend integer references users(id);
alter table notifications add constraint alt_key UNIQUE (userid, notificationdate);