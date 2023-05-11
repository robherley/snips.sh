# Database

Currently, the only supported backend is SQLite. If you'd like to change that, feel free [to contribute](https://github.com/robherley/snips.sh/pulls) :tada:

I chose SQLite since it's simplest starting point for a database, it keeps the data close to the application and overall it's an extremely powerful tool. But don't take my word for it, check out this amazing article from [Ben Johnson at Fly.io](https://fly.io/blog/all-in-on-sqlite-litestream/).

## Schema and Migrations

To prevent breaking behavior across hosted snips.sh instances, [Atlas](https://atlasgo.io/) is used as a schema migration tool to declaratively define the schema and the changes required.

The schema is defined at [`internal/db/schema.hcl`](https://github.com/robherley/snips.sh/blob/main/internal/db/schema.hcl). The schema is embedded with the binary at build time, and migrations will be ran when the application is started.

To change the schema, you will need to edit the schema's HCL. For example, if you wanted to add a nickname attribute to the user's table:

```diff
table "users" {
  schema = schema.main
  column "id" {
    type = text
  }
  column "created_at" {
    type = datetime
  }
  column "updated_at" {
    type = datetime
  }
+  column "nickname" {
+    type = text
+    null = true
+  }
  primary_key {
    columns = [column.id]
  }
  index "idx_users_created_at" {
    columns = [column.created_at]
  }
}
```

To preview the SQL required for the migration, you can use the `script/schema-diff` helper script:

```
$ script/schema-diff
```

```sql
-- Add column "nickname" to table: "users"
ALTER TABLE `users` ADD COLUMN `nickname` text NULL;
```

In a Pull Request, if the `schema.hcl` changes, this'll automatically trigger GitHub Actions to add [the SQL as a comment](https://github.com/robherley/snips.sh/pull/5#issuecomment-1510588137) to the PR.

## Replication and Backups

Since SQLite is a single file on disk, the danger of corrupting/losing a database is quite high. Luckily, it's extremely simple to setup [LiteStream](https://litestream.io/).

Wherever your SQLite file is running, all you need is to [setup a LiteStream](https://litestream.io/guides/) on your host and point it to an S3-compatible object storage. It takes [minutes to setup](https://litestream.io/getting-started/), and then you're good to go :+1:

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
