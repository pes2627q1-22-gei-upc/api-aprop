alter table notifications add column text varchar not null default ' ';
alter table notifications alter column text drop default;