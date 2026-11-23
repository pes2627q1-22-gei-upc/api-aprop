create table if not exists astronomicalevents (
    eventid INTEGER not null references events(id) on delete cascade,
    astronomicalid VARCHAR not null primary key
);