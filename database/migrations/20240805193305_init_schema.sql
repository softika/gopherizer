-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name VARCHAR(255) NOT NULL CHECK (first_name <> ''),
    last_name VARCHAR(255) NOT NULL CHECK (last_name <> ''),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS accounts, profiles, roles, account_roles;
-- +goose StatementEnd
