# Database

snips.sh supports SQLite and PostgreSQL. The backend is inferred from
`SNIPS_DB_URL`. SQLite is the default and accepts a path or SQLite DSN:

```bash
SNIPS_DB_URL=data/snips.db
```

To use PostgreSQL, provide a PostgreSQL URL:

```bash
SNIPS_DB_URL=postgres://user:password@localhost:5432/snips?sslmode=disable
```

`SNIPS_DB_FILEPATH` is deprecated. It remains a compatibility fallback when
`SNIPS_DB_URL` is unset, but emits a warning and is no longer shown in usage.

The application uses a connection pool and automatically applies the migrations
for the selected backend at startup.

## Schema and Migrations

Database migrations are managed using [goose](https://github.com/pressly/goose).
Backend-specific migration files live under `internal/db/sqlite/migrations` and
`internal/db/postgres/migrations` and are embedded in the binary.

### Creating a New Migration

Use `just migrate` to create a new migration file:

```bash
just migrate -s -dir internal/db/sqlite/migrations create add_user_nickname sql
```

Create an equivalent migration in each backend directory and use syntax supported
by that database.

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
just migrate -dir internal/db/sqlite/migrations sqlite3 <db-path> up

# Roll back the last migration
just migrate -dir internal/db/sqlite/migrations sqlite3 <db-path> down

# Check current migration status
just migrate -dir internal/db/sqlite/migrations sqlite3 <db-path> status

# Migrate to a specific version
just migrate -dir internal/db/sqlite/migrations sqlite3 <db-path> up-to 2

# PostgreSQL uses its URL in place of <db-path>
just migrate -dir internal/db/postgres/migrations postgres <postgres-url> status
```

For a full list of goose commands, run:

```bash
just migrate --help
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
