UPDATE profiles
SET
    first_name = $1,
    last_name = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $3
RETURNING id, created_at, updated_at;