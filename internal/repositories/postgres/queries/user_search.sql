-- name: SearchUsers :many
-- Active users whose username or name contains the pattern (already escaped for LIKE), except the requester and the
-- users that block or are blocked by the requester. Exact and prefix username matches go first.
SELECT u.id, u.username, u.name, u.profilepicture
FROM Users u
JOIN AccountStates s ON s.ID = u.AccountState
WHERE s.Name = 'Active'
  AND u.ID <> sqlc.arg(requester)::uuid
  AND (u.Username ILIKE '%' || sqlc.arg(pattern)::text || '%' ESCAPE '\'
       OR u.Name ILIKE '%' || sqlc.arg(pattern)::text || '%' ESCAPE '\')
  AND NOT EXISTS (
      SELECT 1 FROM Blocked b
      WHERE (b.BlockingUser = sqlc.arg(requester)::uuid AND b.BlockedUser = u.ID)
         OR (b.BlockingUser = u.ID AND b.BlockedUser = sqlc.arg(requester)::uuid)
  )
ORDER BY u.Username ILIKE sqlc.arg(pattern)::text ESCAPE '\' DESC,
         u.Username ILIKE sqlc.arg(pattern)::text || '%' ESCAPE '\' DESC,
         lower(u.Username) ASC
LIMIT sqlc.arg(page_size)::int OFFSET sqlc.arg(page_offset)::int;

-- name: CountSearchUsers :one
-- Number of users that SearchUsers finds, without pagination.
SELECT COUNT(*)
FROM Users u
JOIN AccountStates s ON s.ID = u.AccountState
WHERE s.Name = 'Active'
  AND u.ID <> sqlc.arg(requester)::uuid
  AND (u.Username ILIKE '%' || sqlc.arg(pattern)::text || '%' ESCAPE '\'
       OR u.Name ILIKE '%' || sqlc.arg(pattern)::text || '%' ESCAPE '\')
  AND NOT EXISTS (
      SELECT 1 FROM Blocked b
      WHERE (b.BlockingUser = sqlc.arg(requester)::uuid AND b.BlockedUser = u.ID)
         OR (b.BlockingUser = u.ID AND b.BlockedUser = sqlc.arg(requester)::uuid)
  );
