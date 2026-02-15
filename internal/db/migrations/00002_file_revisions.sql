-- +goose Up
CREATE TABLE IF NOT EXISTS `revisions` (
    `id` integer NOT NULL,
    `file_id` text NOT NULL,
    `created_at` datetime NOT NULL,
    `diff` blob NOT NULL,
    `size` integer NOT NULL,
    `type` text NOT NULL,
    PRIMARY KEY (`file_id`, `id`)
);

-- +goose Down
DROP TABLE IF EXISTS `revisions`;
