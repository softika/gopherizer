-- +goose Up
-- +goose StatementBegin
INSERT INTO accounts (id, email, password, created_at, updated_at)
VALUES
    ('a1b2c3d4-1111-2222-3333-444455556666', 'john@mail.com', '$2a$10$.2/hbR6YIEfp4a7zvZ7xpO0fUUySsjM6.wgH0aWuqFN/sJPR5uEFq', NOW(), NOW()),
    ('b1c2d3e4-2222-3333-4444-555566667777', 'jane@mail.com', '$2a$10$.2/hbR6YIEfp4a7zvZ7xpO0fUUySsjM6.wgH0aWuqFN/sJPR5uEFq', NOW(), NOW()),
    ('c1d2e3f4-3333-4444-5555-666677778888', 'alice@mail.com', '$2a$10$.2/hbR6YIEfp4a7zvZ7xpO0fUUySsjM6.wgH0aWuqFN/sJPR5uEFq', NOW(), NOW()),
    ('2f6f112a-a8e2-42c3-a6b0-c15e86d01704', 'milan@mail.com', '$2a$10$.2/hbR6YIEfp4a7zvZ7xpO0fUUySsjM6.wgH0aWuqFN/sJPR5uEFq', NOW(), NOW());

INSERT INTO profiles (id, account_id, first_name, last_name, created_at, updated_at)
VALUES
    ('0dd35f9a-0d20-41f1-80c2-d7993e313fb4', 'a1b2c3d4-1111-2222-3333-444455556666', 'John', 'Doe', NOW(), NOW()),
    ('0dd35f9a-0d20-41f1-80c2-d7993e313fb5', 'b1c2d3e4-2222-3333-4444-555566667777', 'Jane', 'Smith', NOW(), NOW()),
    ('0dd35f9a-0d20-41f1-80c2-d7993e313fb6', 'c1d2e3f4-3333-4444-5555-666677778888', 'Alice', 'Wonderland', NOW(), NOW());

INSERT INTO account_roles (id, account_id, role_id, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'a1b2c3d4-1111-2222-3333-444455556666',
     (SELECT id FROM roles WHERE name = 'ADMIN'), NOW(), NOW()),
    (gen_random_uuid(), 'b1c2d3e4-2222-3333-4444-555566667777',
     (SELECT id FROM roles WHERE name = 'USER'), NOW(), NOW()),
    (gen_random_uuid(), 'c1d2e3f4-3333-4444-5555-666677778888',
     (SELECT id FROM roles WHERE name = 'USER'), NOW(), NOW()),
    (gen_random_uuid(), '2f6f112a-a8e2-42c3-a6b0-c15e86d01704',
     (SELECT id FROM roles WHERE name = 'USER'), NOW(), NOW());
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
TRUNCATE TABLE account_roles, profiles, roles, accounts RESTART IDENTITY CASCADE;
-- +goose StatementEnd
