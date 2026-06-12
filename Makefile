# OpsOne — local dev helpers
# Requires: docker, mysql client (optional), Go 1.21+, Node 20+

MYSQL_HOST ?= localhost
MYSQL_PORT ?= 3306
MYSQL_USER ?= app
MYSQL_PASS ?= secret
MYSQL_DB   ?= opsone
MYSQL_DSN  ?= $(MYSQL_USER):$(MYSQL_PASS)@tcp($(MYSQL_HOST):$(MYSQL_PORT))/$(MYSQL_DB)?parseTime=true

MYSQL_CMD = mysql -h$(MYSQL_HOST) -P$(MYSQL_PORT) -u$(MYSQL_USER) -p$(MYSQL_PASS) $(MYSQL_DB)

.PHONY: db-up db-down migrate seed db-verify run-api run-mock run-agent test

db-up:
	docker compose up -d mysql

db-down:
	docker compose down

migrate: db-up
	@echo "Waiting for MySQL..."
	@sleep 8
	mysql -h$(MYSQL_HOST) -P$(MYSQL_PORT) -uroot -prootsecret < db/schema.sql

seed:
	$(MYSQL_CMD) < db/seed.sql

db-verify:
	$(MYSQL_CMD) -e "SELECT COUNT(*) AS product_count FROM products;"
	$(MYSQL_CMD) -e "SELECT COUNT(*) AS routing_rows FROM routing_config;"

run-api:
	MYSQL_DSN="$(MYSQL_DSN)" go run ./cmd/api

run-mock:
	MYSQL_DSN="$(MYSQL_DSN)" go run ./cmd/worker-mock

run-agent:
	MYSQL_DSN="$(MYSQL_DSN)" go run ./cmd/worker-agent

test:
	go test ./...
