alter table eventratings drop constraint eventratings_rating_check;
alter table eventratings add constraint eventratings_rating_check check (rating >=1 and rating <=5);
alter table userratings add column rating integer not null;
alter table userratings add constraint userratings_rating_check check (rating >=1 and rating <=5);