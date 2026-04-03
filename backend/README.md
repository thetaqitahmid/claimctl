# claimctl Backend

## Working with Database

The database migration is done using `sqlc` and `migrate` tools. The `sqlc` tool
is used to generate Go code from SQL queries and the `migrate` tool is used to
manage the database schema.

First, install `sqlc` and `migrate` tools:

```bash
go get github.com/kyleconroy/sqlc/cmd/sqlc
go get -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate
```

### Database Migration and Schema Management

Database migrations are version-controlled changes to your database schema that
help track and manage database structure modifications over time. Each migration
includes both the changes to apply (up) and their rollback procedures (down),
making it safe to evolve your database schema as your application grows.

To create a new migration, run:

```bash
migrate create -ext sql -dir migrations -seq <MIGRATION_NAME>
# example: migrate create -ext sql -dir migrations -seq create_users_table
```

After that write the SQL queries within migration directory as needed. The
convention is to write a pair of `up` and `down` queries in two separate files
with the same name followed by a version prefix. To apply the migration, run:

```bash
migrate --path migrations -database "<DB CONNECTION>" up
```

### Auto-generating Go code from SQL queries

`sqlc` is used to generate Go code from SQL queries. To generate the Go code, we
need to create a `sqlc.yaml` file in the root directory of the project. The
`sqlc.yaml` file contains the configuration for the `sqlc` tool.

Next, we need to write the queries required for our scenario in the `sqlc`
format in the `sql/queries` directory. The `sqlc` tool will generate the Go code
in the `internal/db` directory.

To generate the Go code, run:

sqlc generate

````

## API Documentation

The API documentation is generated using [Swag](https://github.com/swaggo/swag).

### Prerequisites

Install the `swag` CLI:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
````

### Generating Documentation

To generate or update the Swagger documentation, run the following command from
the `backend` directory:

```bash
swag init -g cmd/main.go
```

This will generate the necessary files in the `docs/` directory.

### Viewing Documentation

Start the backend server:

```bash
make backend_up
```

Then access the Swagger UI at:

http://localhost:8080/swagger/index.html
