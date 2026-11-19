-- name: BlockUser :one
-- Blocks the user; no rows if the block already exists. R2 (no self block) is also checked by the table CHECK.
INSERT INTO Blocked (BlockingUser, BlockedUser, DateBlocked)
VALUES (sqlc.arg(blocker)::uuid, sqlc.arg(blocked)::uuid, now())
ON CONFLICT (BlockingUser, BlockedUser) DO NOTHING
RETURNING DateBlocked;

-- name: RejectActiveFriendRequests :exec
-- R3: no Pending or Accepted friend request may remain between two users when one blocks the other.
UPDATE FriendRequests
SET State = 'Rejected'::friendrequeststate, DateResolved = COALESCE(DateResolved, now())
WHERE State IN ('Pending'::friendrequeststate, 'Accepted'::friendrequeststate)
  AND ((Sender = sqlc.arg(user1)::uuid AND Receiver = sqlc.arg(user2)::uuid)
    OR (Sender = sqlc.arg(user2)::uuid AND Receiver = sqlc.arg(user1)::uuid));

-- name: UnblockUser :execrows
DELETE FROM Blocked
WHERE BlockingUser = sqlc.arg(blocker)::uuid AND BlockedUser = sqlc.arg(blocked)::uuid;

-- name: ListBlockedUsers :many
-- Users blocked by the blocker, the most recent first.
SELECT u.id, u.username, u.name, u.profilepicture, b.DateBlocked
FROM Blocked b
JOIN Users u ON u.ID = b.BlockedUser
WHERE b.BlockingUser = sqlc.arg(blocker)::uuid
ORDER BY b.DateBlocked DESC, lower(u.Username);
