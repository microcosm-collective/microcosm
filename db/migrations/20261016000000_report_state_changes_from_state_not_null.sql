-- +goose Up
-- +goose StatementBegin
DELETE FROM report_state_changes WHERE from_state IS NULL;

ALTER TABLE report_state_changes
    ALTER COLUMN from_state SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE report_state_changes
    ALTER COLUMN from_state DROP NOT NULL;
-- +goose StatementEnd
