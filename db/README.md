# db

it's sqlite

## migrations

i like to use [`golang-migrate/migrate`](https://github.com/golang-migrate/migrate)

for [sqlite specifically](https://github.com/golang-migrate/migrate/tree/master/database/sqlite3), migrations are implicitly wrapped in transactions blocks

run all cmds from the project's root

### generate

```
migrate create -ext sql -dir db/migrations -seq create_foo_bar
```

### up migration

```
go run db/migrate.go
```
