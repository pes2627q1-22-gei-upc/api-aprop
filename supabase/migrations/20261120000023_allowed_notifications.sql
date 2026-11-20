create table allowednotifications (
    Settings integer references Settings(ID) ON DELETE CASCADE,
    NotificationType integer references Notificationtypes(ID) ON DELETE CASCADE,
    primary key (Settings, NotificationType)
);