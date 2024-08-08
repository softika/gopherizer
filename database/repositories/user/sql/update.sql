UPDATE users
SET
    first_name = $1,
    last_name = $2,
    email = $3,
    password = $4,
    enabled = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $6
RETURNING id, created_at, updated_at;