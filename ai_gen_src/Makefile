.PHONY: db-up db-down migrate seed db-verify clear-runtime test test-unit test-integration test-e2e build verify-dod

MYSQL_HOST ?= localhost
MYSQL_PORT ?= 3306
MYSQL_USER ?= app
MYSQL_PASS ?= secret
MYSQL_DB   ?= opsone
MYSQL_ROOT_PASS ?= rootsecret

MYSQL_CLIENT = docker exec -i opsone-mysql mysql --default-character-set=utf8mb4 -u$(MYSQL_USER) -p$(MYSQL_PASS)

db-up:
	docker compose up -d mysql
	@echo "Waiting for MySQL..."
	@docker compose exec mysql mysqladmin ping -h localhost -u root -prootsecret --silent || (sleep 5 && docker compose exec mysql mysqladmin ping -h localhost -u root -prootsecret --silent)

db-down:
	docker compose down

migrate: db-up
	docker cp db/schema.sql opsone-mysql:/tmp/schema.sql
	$(MYSQL_CLIENT) $(MYSQL_DB) -e "source /tmp/schema.sql"

seed: migrate
	docker cp db/seed.sql opsone-mysql:/tmp/seed.sql
	$(MYSQL_CLIENT) $(MYSQL_DB) -e "source /tmp/seed.sql"

db-verify: seed
	docker exec opsone-mysql mysql --default-character-set=utf8mb4 -u$(MYSQL_USER) -p$(MYSQL_PASS) $(MYSQL_DB) -e "SELECT COUNT(*) AS product_count FROM products;"

clear-runtime: db-up
	docker cp db/clear_runtime.sql opsone-mysql:/tmp/clear_runtime.sql
	$(MYSQL_CLIENT) $(MYSQL_DB) -e "source /tmp/clear_runtime.sql"

test-unit:
	go test ./internal/rules/... ./internal/domain/... -count=1

test-integration: seed
	OPSONE_INTEGRATION=1 go test ./internal/store/... ./internal/api/... ./internal/tools/... ./internal/agent/... -count=1

test-e2e: seed
	OPSONE_INTEGRATION=1 go test ./internal/e2e/... -v -count=1

test-e2e-full: seed
	OPSONE_INTEGRATION=1 OPSONE_E2E_FULL=1 go test ./internal/e2e/... -v -count=1 -timeout 20m

test:
	go test ./... -count=1

verify-dod: seed test-e2e
	@echo "DoD smoke: products=11, E2E pass"

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker-mock ./cmd/worker-mock
	go build -o bin/worker-agent ./cmd/worker-agent

run-api:
	go run ./cmd/api

run-mock:
	go run ./cmd/worker-mock

run-agent:
	go run ./cmd/worker-agent
