alter table events alter column StartDate set not null;
alter table events alter column Description set not null;
alter table events alter column Free set not null;
alter table events alter column Permanent set not null;
alter table events drop column Modality;

drop type ModalityType;
alter table events add column Online Boolean NOT NULL default False;
alter table events alter column Online drop default;

CREATE TABLE Places (
    ID Integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    Name Varchar NOT NULL,
    Address Varchar NOT NULL,
    Municipality Varchar,
    Region Varchar,
    Latitude Real,
    Longitude Real,
    PostalCode Varchar,
    UNIQUE (Name, Address),
    CHECK (Latitude <= 90 AND Latitude >= -90),
    CHECK (Longitude <= 180 AND Longitude >= -180)
);

alter table events drop column Place;
alter table events drop column Latitude;
alter table events drop column Longitude;
alter table events drop column PostalCode;
alter table events drop column Municipality;
alter table events drop column Region;
alter table events drop column Address;
alter table events add column PlaceID Integer REFERENCES Places(ID);