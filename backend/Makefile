# Makefile for the Punto Vecinal API (Go + Supabase)
# Run from backend/. The Supabase project lives in ../supabase.

MAIN         := ./cmd/punto-vecinal-api
BIN          := main
SUPABASE_DIR := ..
OAPI_CONFIG  := api/http/oapi-codegen.yaml
OAPI_SPEC    := api/openapi.yaml

.DEFAULT_GOAL := help

# Show available targets
help:
	@echo "Go:"
	@echo "  build            Build the API binary"
	@echo "  run              Run the API"
	@echo "  watch            Live reload with air"
	@echo "  test             Run all tests"
	@echo "  fmt / vet / tidy Format, vet, and tidy modules"
	@echo "  clean            Remove the binary"
	@echo "Code generation:"
	@echo "  generate         Run oapi-codegen and sqlc"
	@echo "  generate-api     HTTP layer from the OpenAPI contract"
	@echo "  generate-db      Database access from Supabase migrations"
	@echo "Supabase (local):"
	@echo "  supabase-start   Start the local stack (needs Docker)"
	@echo "  supabase-stop    Stop the local stack"
	@echo "  supabase-status  Show URLs and keys of the local stack"
	@echo "  db-reset         Recreate the local DB from migrations"
	@echo "  migration-new    Create a migration: make migration-new name=add_reports"
	@echo "  db-lint          Lint the local DB schema"

# --- Go ---

build:
	@echo "Building..."
	@go build -o $(BIN) $(MAIN)

run:
	@go run $(MAIN)

test:
	@echo "Testing..."
	@go test ./... -v

fmt:
	@go fmt ./...

vet:
	@go vet ./...

tidy:
	@go mod tidy

clean:
	@echo "Cleaning..."
	@rm -f $(BIN)

# Live reload
watch:
	@if command -v air > /dev/null; then \
		air; \
	else \
		read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
		if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
			go install github.com/air-verse/air@latest; \
			air; \
		else \
			echo "You chose not to install air. Exiting..."; \
			exit 1; \
		fi; \
	fi

# --- Code generation (tools pinned in go.mod) ---

generate: generate-api generate-db

generate-api:
	@echo "Generating HTTP layer..."
	@go tool oapi-codegen -config $(OAPI_CONFIG) $(OAPI_SPEC)

generate-db:
	@echo "Generating database access..."
	@go tool sqlc generate

# --- Supabase (local stack via the Supabase CLI) ---

supabase-start:
	@supabase start --workdir $(SUPABASE_DIR)

supabase-stop:
	@supabase stop --workdir $(SUPABASE_DIR)

supabase-status:
	@supabase status --workdir $(SUPABASE_DIR)

# Drops the local DB and reapplies every migration in supabase/migrations
db-reset:
	@supabase db reset --workdir $(SUPABASE_DIR)

migration-new:
	@test -n "$(name)" || { echo "Usage: make migration-new name=<migration_name>"; exit 1; }
	@supabase migration new $(name) --workdir $(SUPABASE_DIR)

db-lint:
	@supabase db lint --workdir $(SUPABASE_DIR)

.PHONY: help build run test fmt vet tidy clean watch \
	generate generate-api generate-db \
	supabase-start supabase-stop supabase-status db-reset migration-new db-lint
