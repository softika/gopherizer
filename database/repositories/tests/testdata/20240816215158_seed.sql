-- +goose Up
-- +goose StatementBegin
INSERT INTO profiles (first_name, last_name, email)
VALUES ('John', 'Smith', 'john@email.com'),
       ('Jane', 'Doe', 'jane@email.com'),
       ('Alice', 'Wonderland', 'alice@email.com')
ON CONFLICT DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
TRUNCATE profiles CASCADE;
-- +goose StatementEnd
