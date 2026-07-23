# Simila API

Simila is a Go API for rating films and shows, discovering media, reacting to ratings, and finding user matches.

This project is currently in development. APIs, schemas, and workflows may change as the product is built out.

## Requirements

- Go
- PostgreSQL
- `goose` for database migrations
- `sqlc` for query code generation

## Setup

Create a `.env` file with the required environment variables:

```sh
DB_URL=postgres://user:password@localhost:5432/simila?sslmode=disable
TMDB_TOKEN=your_tmdb_token
CORS_ALLOWED_ORIGINS=http://localhost:3000
```

Apply migrations:

```sh
make db/migrations/up
```

Regenerate sqlc code after changing SQL queries:

```sh
make sqlc/generate
```

Run the API:

```sh
make run/api
```

Run tests:

```sh
go test ./...
```
