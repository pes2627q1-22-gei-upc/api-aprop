alter table eventratings drop constraint eventratings_rating_check;
alter table eventratings add constraint eventratings_rating_check check (rating >=0 and rating <=5);
alter table userratings drop column rating;