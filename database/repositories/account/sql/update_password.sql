UPDATE accounts
SET password = $1, updated_at = CURRENT_TIMESTAMP
WHERE id = $2