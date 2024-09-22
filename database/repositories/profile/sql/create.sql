INSERT INTO profiles (account_id, first_name, last_name, created_at, updated_at)
VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
RETURNING id, created_at, updated_at;