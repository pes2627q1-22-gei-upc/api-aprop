alter table notifications alter column type drop not null;

delete from notifications;
alter table notifications drop constraint notifications_pkey;
alter table notifications drop column id;
alter table notifications drop constraint alt_key;
alter table notifications drop column userid;
alter table notifications add column userid uuid;
alter table notifications drop column friend;
alter table notifications add column friend uuid;
alter table notifications add constraint notifications_pkey primary key (userid, notificationdate);
alter table notifications drop column read;