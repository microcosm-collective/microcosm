-- +goose Up
-- +goose StatementBegin
ALTER TABLE reporters ADD COLUMN note text;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE reporters DROP COLUMN note;
-- +goose StatementEnd
