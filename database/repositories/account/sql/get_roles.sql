SELECT name FROM account_roles AS ar
INNER JOIN roles AS r ON ar.role_id = r.id
WHERE ar.account_id = $1;