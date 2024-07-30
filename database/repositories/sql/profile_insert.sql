INSERT INTO profiles (first_name, last_name, created_at, updated_at)
VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
RETURNING id, created_at, updated_at;