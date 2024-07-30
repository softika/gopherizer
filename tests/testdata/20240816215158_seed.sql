-- +goose Up
-- +goose StatementBegin
INSERT INTO profiles (id, first_name, last_name, created_at, updated_at)
VALUES
    ('0dd35f9a-0d20-41f1-80c2-d7993e313fb4', 'John', 'Doe', NOW(), NOW()),
    ('0dd35f9a-0d20-41f1-80c2-d7993e313fb5', 'Jane', 'Smith', NOW(), NOW()),
    ('0dd35f9a-0d20-41f1-80c2-d7993e313fb6', 'Alice', 'Wonderland', NOW(), NOW()),
    ('0dd35f9a-0d20-41f1-80c2-d7993e313fb7', 'Bob', 'Builder', NOW(), NOW());
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
TRUNCATE TABLE profiles RESTART IDENTITY CASCADE;
-- +goose StatementEnd
