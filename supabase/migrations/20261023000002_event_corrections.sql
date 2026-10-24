alter table events alter column StartDate drop not null;
alter table events alter column Description drop not null;
alter table events alter column Free drop not null;
alter table events alter column Permanent drop not null;
alter table events drop column Online;

create type ModalityType as enum ('Online', 'Face-to-Face', 'Online and Face-to-Face');
alter table events add column Modality ModalityType;

alter table events drop column PlaceID;
alter table events add column Place Varchar;
alter table events add column Latitude Real;
alter table events add column Longitude Real;
alter table events add column PostalCode Varchar;
alter table events add column Municipality Varchar;
alter table events add column Region Varchar;
alter table events add column Address Varchar;
alter table events add constraint valid_lat CHECK (Latitude <= 90 AND Latitude >= -90);
alter table events add constraint valid_long CHECK (Longitude <= 180 AND Longitude >= -180);
drop table places