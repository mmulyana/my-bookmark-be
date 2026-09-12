.PHONY: dev build migrate-up migrate-down sqlc generate docker-up docker-down

# Build binary
build:
	go build -o bin/server ./cmd/server

# Run with hot reload (requires: go install github.com/air-verse/air@latest)
dev:
	air

# Run directly without hot reload
run:
	go run ./cmd/server

# Database migrations
migrate-up:
	migrate -database "$(DATABASE_URL)" -path db/migrations up

migrate-down:
	migrate -database "$(DATABASE_URL)" -path db/migrations down 1

# Generate sqlc code (requires sqlc installed)
sqlc:
	sqlc generate -f db/sqlc.yaml

# Generate OpenAPI server code (requires oapi-codegen installed)
generate:
	oapi-codegen -config api/oapi-codegen.yaml api/openapi.yaml

# Docker
docker-up:
	docker compose up -d

docker-down:
	docker compose down

# Go tools
vet:
	go vet ./...

tidy:
	go mod tidy
