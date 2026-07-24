-- +goose Up
-- +goose StatementBegin
ALTER TABLE `users` ADD COLUMN `theme_color` text NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `users` DROP COLUMN `theme_color`;
-- +goose StatementEnd
