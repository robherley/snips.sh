-- +goose Up
-- +goose StatementBegin
ALTER TABLE `files` ADD COLUMN `name` text;

-- names are unique per user (case-insensitively); unnamed files are NULL and
-- exempt from the index
CREATE UNIQUE INDEX IF NOT EXISTS `idx_files_user_id_name` ON `files` (`user_id`, `name` COLLATE NOCASE) WHERE `name` IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS `idx_files_user_id_name`;

ALTER TABLE `files` DROP COLUMN `name`;
-- +goose StatementEnd
