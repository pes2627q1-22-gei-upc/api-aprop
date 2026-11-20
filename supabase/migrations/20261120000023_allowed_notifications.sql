create table if not exists allowednotifications (
    Settings integer references Settings(ID) ON DELETE CASCADE,
    NotificationType integer references Notificationtypes(ID) ON DELETE CASCADE,
    primary key (Settings, NotificationType)
) enable row level security;

insert into allowednotifications (settings, notificationtype) select s.id, nt.id from settings s, notificationtypes nt where not exists (select * from allowednotifications an where an.settings=s.id);