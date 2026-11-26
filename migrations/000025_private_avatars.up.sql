-- #247: profile pictures now live in a private Supabase Storage bucket and Users.ProfilePicture stores the object
-- path (<user id>/avatar-<unix ms>.<ext>). Until now, without S3_ENDPOINT, POST /users/me/avatar uploaded nothing and
-- stored a URL of a nonexistent object (https://<bucket>.s3.<region>.amazonaws.com/avatars/...): clear those URLs.
UPDATE Users SET ProfilePicture = NULL
WHERE ProfilePicture ~ '^https://[^/]+\.s3\.[^/]+\.amazonaws\.com/avatars/';
