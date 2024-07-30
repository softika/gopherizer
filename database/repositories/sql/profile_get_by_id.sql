SELECT p.id, p.first_name, p.last_name, p.created_at, p.updated_at
FROM profiles AS p
WHERE p.id = $1;