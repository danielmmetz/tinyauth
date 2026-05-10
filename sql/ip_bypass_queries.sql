-- name: CreateIPBypass :one
INSERT INTO "ip_bypasses" (
    "cidr",
    "domain",
    "expires_at",
    "note",
    "created_by",
    "created_at"
) VALUES (
    ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: ListActiveIPBypassesForDomain :many
SELECT * FROM "ip_bypasses"
WHERE ("domain" = ? OR "domain" = '*') AND "expires_at" > ?;

-- name: ListIPBypasses :many
SELECT * FROM "ip_bypasses"
ORDER BY "created_at" DESC;

-- name: ListIPBypassesByCreator :many
SELECT * FROM "ip_bypasses"
WHERE "created_by" = ?
ORDER BY "created_at" DESC;

-- name: DeleteIPBypass :exec
DELETE FROM "ip_bypasses"
WHERE "id" = ?;

-- name: DeleteIPBypassForCreator :execrows
DELETE FROM "ip_bypasses"
WHERE "id" = ? AND "created_by" = ?;

-- name: DeleteExpiredIPBypasses :execrows
DELETE FROM "ip_bypasses"
WHERE "expires_at" < ?;
