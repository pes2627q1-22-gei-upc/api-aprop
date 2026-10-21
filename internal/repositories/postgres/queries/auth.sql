-- name: GetUserByFirebaseUID :one
SELECT * FROM Users
WHERE FirebaseUID = sqlc.arg(firebase_uid)::varchar;

-- name: CreateUser :one
-- Creates the user together with its settings row in a single statement, so a failed insert
-- (e.g. username already taken) does not leave orphan settings behind.
WITH new_settings AS (
    INSERT INTO Settings DEFAULT VALUES
    RETURNING ID
)
INSERT INTO Users (FirebaseUID, Username, Email, AccountState, Settings)
SELECT
    sqlc.arg(firebase_uid)::varchar, sqlc.arg(username)::varchar, sqlc.narg(email)::varchar,
    (SELECT ID FROM AccountStates WHERE Name = 'Active'), new_settings.ID
FROM new_settings
RETURNING *;

-- name: UpdateUserEmail :one
UPDATE Users SET Email = sqlc.narg(email)::varchar
WHERE ID = sqlc.arg(id)::integer
RETURNING *;
