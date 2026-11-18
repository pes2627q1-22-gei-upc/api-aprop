-- Queries used by UserRepository.DeleteUser to erase a user and every row that references it (RGPD).
-- They run in a single transaction, in an order that respects the foreign keys.

-- name: ListCreatedUnofficialEventIDs :many
SELECT ID FROM UnofficialEvents WHERE Creator = sqlc.arg(user_id)::uuid ORDER BY ID;

-- name: DeleteUserNotifications :exec
-- Notifications of the user and the ones of other users about the user (friend requests).
DELETE FROM Notifications WHERE UserID = sqlc.arg(user_id)::uuid OR Friend = sqlc.arg(user_id)::uuid;

-- name: DeleteUserFriendRequests :exec
DELETE FROM FriendRequests WHERE Sender = sqlc.arg(user_id)::uuid OR Receiver = sqlc.arg(user_id)::uuid;

-- name: DeleteUserFriendships :exec
DELETE FROM Friendships WHERE User1 = sqlc.arg(user_id)::uuid OR User2 = sqlc.arg(user_id)::uuid;

-- name: DeleteUserBlocks :exec
DELETE FROM Blocked WHERE BlockingUser = sqlc.arg(user_id)::uuid OR BlockedUser = sqlc.arg(user_id)::uuid;

-- name: DeleteUserSignUps :exec
DELETE FROM SignUp WHERE UserID = sqlc.arg(user_id)::uuid;

-- name: DeleteUserEventRatings :exec
DELETE FROM EventRatings WHERE UserID = sqlc.arg(user_id)::uuid;

-- name: DeleteUserUserRatings :exec
-- Ratings given and received by the user.
DELETE FROM UserRatings WHERE Rater = sqlc.arg(user_id)::uuid OR Rated = sqlc.arg(user_id)::uuid;

-- name: DeleteUserMessages :exec
DELETE FROM Messages WHERE UserID = sqlc.arg(user_id)::uuid;

-- name: DeleteUserPreferenceTestChoices :exec
DELETE FROM PreferenceTestChoices WHERE TestUser = sqlc.arg(user_id)::uuid;

-- name: DeleteUserPreferenceTests :exec
DELETE FROM PreferenceTests WHERE UserID = sqlc.arg(user_id)::uuid;

-- name: DeleteUserPreferences :exec
DELETE FROM UserPreferences WHERE UserID = sqlc.arg(user_id)::uuid;

-- name: DeleteUserLevel :exec
DELETE FROM UserLevels WHERE UserID = sqlc.arg(user_id)::uuid;

-- name: DeleteUserState :exec
DELETE FROM UserState WHERE UserID = sqlc.arg(user_id)::uuid;

-- name: DeleteUserChatbotConversations :exec
-- Messages, recommendations and feedback are removed by ON DELETE CASCADE.
DELETE FROM ChatbotConversations WHERE UserID = sqlc.arg(user_id)::uuid;

-- name: DeleteUserRow :one
-- Returns the settings row of the deleted user, so it can be deleted afterwards.
DELETE FROM Users WHERE ID = sqlc.arg(user_id)::uuid RETURNING Settings;

-- name: DeleteSettings :exec
DELETE FROM Settings WHERE ID = sqlc.arg(id)::integer;
