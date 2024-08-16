-- +goose Up
-- +goose StatementBegin
INSERT INTO users (first_name, last_name, email, password)
VALUES ('John', 'Smith', 'john@email.com', 'password123!'),
       ('Jane', 'Doe', 'jane@email.com', 'password123!'),
       ('Alice', 'Wonderland', 'alice@email.com', 'password123!')
ON CONFLICT DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
TRUNCATE users CASCADE;
-- +goose StatementEnd
