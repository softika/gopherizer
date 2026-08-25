-- Switch the profiles primary key default from UUIDv4 to UUIDv7.
--
-- v4 is random, so inserts land at random points in the primary key's B-tree:
-- every insert dirties a different page and the index fragments as the table
-- grows. v7 embeds a millisecond timestamp in its high bits, so new rows append
-- to the right-hand edge of the index instead.
--
-- Requires PostgreSQL 18 or newer, which is where uuidv7() was added. Existing
-- rows keep their v4 identifiers; both versions are valid values of the uuid
-- type and mix freely in the same column.

-- +goose Up
-- +goose StatementBegin
ALTER TABLE profiles ALTER COLUMN id SET DEFAULT uuidv7();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE profiles ALTER COLUMN id SET DEFAULT gen_random_uuid();
-- +goose StatementEnd
