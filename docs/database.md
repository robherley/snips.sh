# Database

Currently, the only supported backend is SQLite. If you'd like to change that, feel free [to contribute](https://github.com/robherley/snips.sh/pulls) :tada:

I chose SQLite since it's simplest starting point for a database, it keeps the data close to the application and overall it's an extremely powerful tool. But don't take my word for it, check out this amazing article from [Ben Johnson at Fly.io](https://fly.io/blog/all-in-on-sqlite-litestream/).

## Schema and Migrations

Database migrations are managed using [goose](https://github.com/pressly/goose). Migration files are located in [`internal/db/migrations/`](https://github.com/robherley/snips.sh/tree/main/internal/db/migrations) and are embedded with the binary at build time. Migrations are automatically applied when the application starts.

### Creating a New Migration

Use `script/migrator` to create a new migration file:

```bash
script/migrator -s -dir internal/db/migrations create add_user_nickname sql
```

This creates a new file like `internal/db/migrations/00002_add_user_nickname.sql` with the goose annotations. Edit it to add your up and down SQL:

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE `users` ADD COLUMN `nickname` text NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `users` DROP COLUMN `nickname`;
-- +goose StatementEnd
```

### Running Migrations Manually

To run migrations manually via CLI:

```bash
# Apply all pending migrations
script/migrator -dir internal/db/migrations sqlite3 <db-path> up

# Roll back the last migration
script/migrator -dir internal/db/migrations sqlite3 <db-path> down

# Check current migration status
script/migrator -dir internal/db/migrations sqlite3 <db-path> status

# Migrate to a specific version
script/migrator -dir internal/db/migrations sqlite3 <db-path> up-to 2
```

For a full list of goose commands, run:

```bash
script/migrator --help
```

## Replication and Backups

Since SQLite is a single file on disk, the danger of corrupting/losing a database is quite high. Luckily, it's extremely simple to set up [LiteStream](https://litestream.io/).

Wherever your SQLite file is running, all you need is to [set up a LiteStream](https://litestream.io/guides/) on your host and point it to an S3-compatible object storage. It takes [minutes to set up](https://litestream.io/getting-started/), and then you're good to go :+1:

Here's an example of a `docker-compose.yml`:

```yaml
version: "3"
services:
  litestream:
    command: replicate
    image: 'litestream/litestream'
    restart: unless-stopped
    volumes:
      - /home/snips/data:/data
      - ./litestream.yml:/etc/litestream.yml
```

And the `litestream.yml` configuration:

```yaml
access-key-id: <secret>
secret-access-key: <secret>

dbs:
  - path: /data/snips.db
    replicas:
      - url: s3://<url>/backups
```
