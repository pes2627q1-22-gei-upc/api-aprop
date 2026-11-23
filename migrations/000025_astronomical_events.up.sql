create table if not exists astronomicalevents (
    eventid INTEGER not null,
    astronomicalid VARCHAR not null primary key,
    foreign key (eventid) references events(id) on delete cascade
);