-- +goose Up
-- +goose StatementBegin
CREATE TABLE `api_keys` (
	`id` text PRIMARY KEY,
	`created_at` datetime,
	`updated_at` datetime,
	`name` text,
	`token_hash` text NOT NULL,
	`user_id` text NOT NULL,
	`last_used_at` datetime,
	`expires_at` datetime
);

CREATE UNIQUE INDEX `idx_api_keys_token_hash` ON `api_keys` (`token_hash`);

CREATE INDEX `idx_api_keys_user_id` ON `api_keys` (`user_id`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE `api_keys`;
-- +goose StatementEnd
