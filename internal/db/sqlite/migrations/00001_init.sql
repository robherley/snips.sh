-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `users` (
    `id` text NOT NULL,
    `created_at` datetime NOT NULL,
    `updated_at` datetime NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE INDEX IF NOT EXISTS `idx_users_created_at` ON `users` (`created_at`);

CREATE TABLE IF NOT EXISTS `public_keys` (
    `id` text NOT NULL,
    `created_at` datetime NOT NULL,
    `updated_at` datetime NOT NULL,
    `fingerprint` text NOT NULL,
    `type` text NOT NULL,
    `user_id` text NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE INDEX IF NOT EXISTS `idx_public_keys_created_at` ON `public_keys` (`created_at`);
CREATE UNIQUE INDEX IF NOT EXISTS `idx_pubkey_fingerprint` ON `public_keys` (`fingerprint`);

CREATE TABLE IF NOT EXISTS `files` (
    `id` text NOT NULL,
    `created_at` datetime NOT NULL,
    `updated_at` datetime NOT NULL,
    `size` integer NOT NULL,
    `content` blob NOT NULL,
    `private` numeric NOT NULL,
    `type` text NOT NULL,
    `user_id` text NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE INDEX IF NOT EXISTS `idx_files_created_at` ON `files` (`created_at`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `files`;
DROP TABLE IF EXISTS `public_keys`;
DROP TABLE IF EXISTS `users`;
-- +goose StatementEnd
