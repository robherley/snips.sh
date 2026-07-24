-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `revisions` (
    `id` text NOT NULL,
    `sequence` integer NOT NULL,
    `file_id` text NOT NULL,
    `created_at` datetime NOT NULL,
    `diff` blob NOT NULL,
    `size` integer NOT NULL,
    `type` text NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE INDEX IF NOT EXISTS `idx_revisions_file_id_sequence` ON `revisions` (`file_id`, `sequence`);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS `revisions`;
