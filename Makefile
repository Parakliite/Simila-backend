ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]


# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	@go run ./cmd/api

## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	@psql ${DB_URL}

## build: build the api application
.PHONY: build/api
build:
	@go build -ldflags='-s' -o ./bin ./cmd/api 

## db/migrations/up: apply all up database migrations
.PHONY: db/migrations/up 
db/migrations/up: confirm
	@echo "running up migrations"
	@goose -dir ./sql/schema postgres ${DB_URL} up

## sqlc/generate: converts sql queries to Go code
.PHONY: sqlc/generate
sqlc/generate:
	@sqlc generate

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #
## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...
