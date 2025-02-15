PROJECT_DIR = $(shell pwd)
PROJECT_BIN = $(PROJECT_DIR)/bin
$(shell [ -f bin ] || mkdir -p $(PROJECT_BIN))
PATH := $(PROJECT_BIN):$(PATH)

.DEFAULT_GOAL := help

VERSION=$(shell git describe --tags --always --dirty)
.PHONY: build
build: ## build with version
	go build -ldflags "-X clean-arch-template/version.VERSION=$(VERSION)" ./cmd/...

.PHONY: dc
dc: ## run all services using docker compose
	docker-compose up --remove-orphans --build

.PHONY: cleandc
cleandc: ## run all services using docker compose, without postgres volume
	rm -rf pg_volume
	docker-compose up --remove-orphans --build

.PHONY: postgres-init
postgres-init: ## run docker container with postgres db
	docker run --name postgres -p 5433:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=admin -d postgres:15-alpine

.PHONY: postgres-drop
postgres-drop: ## remove docker container with postgres db
	docker stop postgres
	docker remove postgres

.PHONY: postgres
postgres: ## run psql in docker container
	docker exec -it postgres psql

.PHONY: create-db
create-db: ## create db in docker container
	docker exec -it postgres createdb --username=postgres --owner=postgres demo

.PHONY: drop-db
drop-db: ## drop db in docker container
	docker exec -it postgres dropdb demo

.PHONY: docker
docker: ## run application in docker container
	docker build -t template .
	docker run --rm \
		--name template \
		--network host \
		-p 9000:9000 \
		-e DB_PASSWORD=$(DB_PASSWORD) \
		template

# ----------------------------------- TESTING -----------------------------------
.PHONY: tests
tests: ## run tests, excluding integration tests
	go test -count=1 $(shell go list ./... | grep -v integration-test) -v -test.v

.PHONY: test-coverage
test-coverage: ## run code test coverage
	go test -count=1 -race -coverprofile=coverage.out $(shell go list ./... | grep -v integration-test)
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out
# ---------------------------------- PROFILING ----------------------------------
.PHONY: cpuprof
cpuprof: ## run cpu profiling
	( PPROF_TMPDIR=${PPROFDIR} go tool pprof -http :8081 -seconds 20 http://127.0.0.1:9000/debug/pprof/profile )

.PHONY: memprof
memprof: ## run memory profiling
	( PPROF_TMPDIR=${PPROFDIR} go tool pprof -http :8081 http://127.0.0.1:9000/debug/pprof/heap )

# ---------------------------------- LINTING ------------------------------------
GOLANGCI_LINT_VERSION = v1.60.3
GOLANGCI_LINT = $(PROJECT_BIN)/golangci-lint

.PHONY: .install-golangci-lint
.install-golangci-lint:
	[ -f $(PROJECT_BIN)/golangci-lint ] || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(PROJECT_BIN) $(GOLANGCI_LINT_VERSION)

.PHONY: lint
lint: .install-golangci-lint  ## run linter
	gofumpt -w ./..
	$(GOLANGCI_LINT) run ./... --config=./.golangci.yml

.PHONY: lint-fast
lint-fast: .install-golangci-lint ## run fast linter
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
new-migration: .install-migrate ## run migrations
	$(MIGRATE) create -ext sql -dir ./migrations $(name)

# ----------------------------------- KUBERNETES -----------------------------------
.PHONY: k8s-deploy
k8s-deploy: ## Deploy application to Minikube
	eval $$(minikube docker-env) && \
	docker build -t clean-arch-template:latest . && \
	kubectl apply -f k8s.yaml

.PHONY: k8s-deploy-env
k8s-deploy-env: ## Deploy application to Minikube with custom environment variables
	@if [ -f .env ]; then \
		echo "Creating ConfigMap and Secret from .env file..."; \
		kubectl create configmap app-config --from-env-file=.env -o yaml --dry-run=client | kubectl apply -f -; \
		kubectl create secret generic app-secrets --from-env-file=.env -o yaml --dry-run=client | kubectl apply -f -; \
	else \
		echo ".env file not found, using default values from k8s.yaml"; \
	fi
	eval $$(minikube docker-env) && \
	docker build -t clean-arch-template:latest . && \
	kubectl apply -f k8s.yaml

.PHONY: k8s-delete
k8s-delete: ## Delete application from Minikube
	kubectl delete -f k8s.yaml

.PHONY: k8s-status
k8s-status: ## Check application status in Minikube
	@echo "Checking pods status..."
	kubectl get pods -l app=clean-arch-template
	@echo "\nChecking service status..."
	kubectl get services clean-arch-template
	@echo "\nApplication URL:"
	minikube service clean-arch-template --url

.PHONY: k8s-logs
k8s-logs: ## View application logs
	kubectl logs -f deployment/clean-arch-template

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
