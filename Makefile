PROJECT_DIR = $(shell pwd)
PROJECT_BIN = $(PROJECT_DIR)/bin
$(shell [ -f bin ] || mkdir -p $(PROJECT_BIN))
PATH := $(PROJECT_BIN):$(PATH)

.PHONY: dc
dc:
	docker-compose up --remove-orphans --build

.PHONY: cleandc
cleandc:
	rm -rf pg_volume
	docker-compose up --remove-orphans --build

.PHONY: postgres-init
postgres-init:
	docker run --name postgres -p 5433:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=admin -d postgres:15-alpine

.PHONY: postgres-drop
postgres-drop:
	docker stop postgres
	docker remove postgres

.PHONY: postgres
postgres:
	docker exec -it postgres psql

.PHONY: create-db
create-db:
	docker exec -it postgres createdb --username=postgres --owner=postgres demo

.PHONY: drop-db
drop-db:
	docker exec -it postgres dropdb demo

.PHONY: docker
docker:
	docker build -t template .
	docker run --rm \
		--name template \
		--network host \
		-p 9000:9000 \
		-e DB_PASSWORD=$(DB_PASSWORD) \
		template

# ----------------------------------- TESTING -----------------------------------
.PHONY: tests
tests:
	go test -count=1 $(shell go list ./... | grep -v integration-test) -v -test.v

.PHONY: test-coverage
test-coverage:
	go test -count=1 -race -coverprofile=coverage.out $(shell go list ./... | grep -v integration-test)
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out
# ---------------------------------- PROFILING ----------------------------------
.PHONY: cpuprof
cpuprof:
	( PPROF_TMPDIR=${PPROFDIR} go tool pprof -http :8081 -seconds 20 http://127.0.0.1:9000/debug/pprof/profile )

.PHONY: memprof
memprof:
	( PPROF_TMPDIR=${PPROFDIR} go tool pprof -http :8081 http://127.0.0.1:9000/debug/pprof/heap )

# ---------------------------------- LINTING ------------------------------------
GOLANGCI_LINT_VERSION = v1.60.3
GOLANGCI_LINT = $(PROJECT_BIN)/golangci-lint

.PHONY: .install-golangci-lint
.install-golangci-lint:
	[ -f $(PROJECT_BIN)/golangci-lint ] || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(PROJECT_BIN) $(GOLANGCI_LINT_VERSION)

.PHONY: lint
lint: .install-golangci-lint
	gofumpt -w ./..
	$(GOLANGCI_LINT) run ./... --config=./.golangci.yml

.PHONY: lint-fast
lint-fast: .install-golangci-lint
	gofumpt -w ./..
	$(GOLANGCI_LINT) run ./... --fast --config=./.golangci.yml

# ---------------------------------- MIGRATIONS ---------------------------------
MIGRATE_VERSION = 4.17.1
MIGRATE = $(PROJECT_BIN)/migrate

.PHONY: .install-migrate
.install-migrate:
	@if [ ! -f $(MIGRATE) ]; then \
		git clone https://github.com/golang-migrate/migrate.git ./.tmp;  \
		cd ./.tmp/cmd/migrate; \
		git checkout v$(MIGRATE_VERSION); \
		go build; \
		mv migrate* $(PROJECT_BIN); \
		cd $(PROJECT_DIR); \
		sleep 1; \
		rm -rf .tmp; \
	fi

.PHONY: new-migration
new-migration: .install-migrate
	$(MIGRATE) create -ext sql -dir ./migrations $(name)
