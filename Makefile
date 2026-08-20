ifneq (,$(wildcard .env))
include .env
export YP_PSQL_DSN
export YP_CUSTOM_TEST
export DOCKER_COMPOSE/check
endif

SHELL := /bin/bash
PROJECT_DIR ?= $(CURDIR)
TMPDIR ?= /tmp

# Конфигурация
YP_PSQL_DSN ?= ""

GO_PACKAGES := $(shell go list ./... | grep -vE '/mocks|/e2e')
OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')

ifeq ($(OS),darwin)
    ARCH := $(shell uname -m)
else
    ARCH := amd64
endif

# Инструменты
GOLANGCI_LINT ?= golangci-lint
GO ?= go
GIT := git
GREP := grep
RM := rm -f
RMDIR := rm -rf
CD := cd
TAIL := tail
KILL := kill
SLEEP := sleep
DATE := date
DOCKER := docker
DOCKER_COMPOSE ?= $(DOCKER) compose
ECHO := echo -e

NOECHO := @

TAIL_LAST_N_LINES ?= 10

# Логи и PID
SERVER_LOG_FILE := $(TMPDIR)/gophprofile-server.log
SERVER_PID_FILE := $(TMPDIR)/gophprofile-server.pid
SERVER_CMD := $(PROJECT_DIR)/cmd/server
SERVER_BIN := $(SERVER_CMD)/server
SERVER_HOST ?= localhost
SERVER_PORT ?= 9082

WORKER_CMD := $(PROJECT_DIR)/cmd/worker
WORKER_BIN := $(WORKER_CMD)/worker

GO_COVERAGE_REPORT := $(TMPDIR)/gophprofile-coverage.out

GOPHPROFILE_VERSION ?= "dev"

BUILD_VERSION := $(GOPHPROFILE_VERSION)
BUILD_DATE    := $(shell $(DATE) -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_COMMIT  := $(shell $(GIT) rev-parse HEAD)

GO_LDFLAGS := -X main.buildVersion=$(BUILD_VERSION) \
              -X main.buildDate=$(BUILD_DATE) \
              -X main.buildCommit=$(BUILD_COMMIT)

print_title = $(ECHO) "\033[1;33m$1\033[0m"

.PHONY: all help clean \
    build build-server build-worker build-docker \
    test test-go test-e2e \
    lint lint-vet lint-golangci lint-golangci-fix \
    coverage coverage-html \
    clone-yp-autotest \
    check-binaries \
    log-server log-worker \
    start-docker stop-docker \
    start stop restart status

.DEFAULT_GOAL := all

.ONESHELL:

all: stop clean build lint test-go

help: # Shows help message
    $(NOECHO) $(GREP) -E '^[a-zA-Z0-9 -]+:.*#'  Makefile | \
    sort | \
    while read -r l; do \
        printf "\033[1;33m$$(echo $$l | cut -f 1 -d':')\033[00m:$$(echo $$l | cut -f 2- -d'#')\n"; \
    done

build: build-docker # Builds server and worker binaries via docker

build-docker: # Builds docker compose
	$(NOECHO) $(DOCKER_COMPOSE) -f docker-compose.yml build

build-server: # Builds server's binary
	$(NOECHO) $(call print_title,"Building server binary")
	$(NOECHO) $(GO) build -ldflags "$(GO_LDFLAGS)" -o $(SERVER_BIN) $(SERVER_CMD)

build-worker: # Builds worker's binary
	$(NOECHO) $(call print_title,"Building worker binary")
	$(NOECHO) $(GO) build -ldflags "$(GO_LDFLAGS)" -o $(WORKER_BIN) $(WORKER_CMD)

clean: # Removes binaries and logs
	$(NOECHO) $(call print_title,"Removing built binaries and checkouted tests")
	$(NOECHO) $(RM) $(SERVER_BIN) $(WORKER_BIN) \
		$(SERVER_LOG_FILE) \
		$(GO_COVERAGE_REPORT)

lint: lint-vet lint-golangci # Runs linters from govet and golangci-lint respectively

lint-vet: # Runs go vet with structtag check
	$(NOECHO) $(call print_title,"Running go vet with structtag check")
	$(NOECHO) $(GO) vet -structtag ./...

lint-golangci: # Runs linters from golangci-lint
	$(NOECHO) $(call print_title,"Running golangci linters")
	$(NOECHO) $(GOLANGCI_LINT) run

lint-golangci-fix: # Runs golangci-lint with auto-fix
	$(NOECHO) $(GOLANGCI_LINT) run --fix

check-binaries: # Checks the existance of required binaries
	$(NOECHO) $(call print_title,"Looking up for binaries")
	$(NOECHO) if [ ! -f $(SERVER_BIN) -o ! -f $(WORKER_BIN) ]; then \
		$(ECHO) "server or worker binaries were not found"; \
		exit 1; \
	fi

test: test-go test-e2e # Runs tests

test-go: # Runs golang tests
	$(NOECHO) $(call print_title,"Running tests: golang")
	$(NOECHO) $(GO) clean -testcache
	$(NOECHO) $(GO) test -buildvcs=false -v -race -cover ./...

test-e2e: # Runs end2end tests
	$(NOECHO) $(call print_title,"Running e2e tests")
	$(NOECHO) $(GO) test -buildvcs=false -v -tags=e2e ./tests/...

coverage: # Runs tests and shows total coverage
	$(NOECHO) $(call print_title,"Running tests with coverage")
	$(NOECHO) $(GO) test -buildvcs=false -v -race -coverprofile=$(GO_COVERAGE_REPORT) $(GO_PACKAGES)
	$(NOECHO) $(GO) tool cover -func=$(GO_COVERAGE_REPORT)

coverage-html: # Generates HTML coverage report and opens it
	$(NOECHO) $(call print_title,"Generating HTML coverage report")
	$(NOECHO) $(GO) test -v -race -coverprofile=$(GO_COVERAGE_REPORT) $(GO_PACKAGES)
	$(NOECHO) $(GO) tool cover -html=$(GO_COVERAGE_REPORT)

start: # Starts the docker compose infrastructure
	$(NOECHO) $(MAKE) start-docker
    
start-docker: # Starts the Docker container with infrastructure (DB, Kafka, MinIO)
	$(NOECHO) $(call print_title,"Starting up the docker compose containers")
	$(NOECHO) $(DOCKER_COMPOSE) up -d

stop: stop-docker # Stops docker containers

stop-docker: # Stops the Docker container
	$(NOECHO) $(call print_title,"Stopping the docker compose containers")
	$(NOECHO) $(DOCKER_COMPOSE) down

restart: stop start # Restarts services

log-server: # Shows log from server
	$(NOECHO) $(call print_title,"Listing rows from server logs")
	$(NOECHO) $(DOCKER_COMPOSE) logs -f -n $(TAIL_LAST_N_LINES) server

log-worker: # Shows log from worker
	$(NOECHO) $(call print_title,"Listing rows from worker logs")
	$(NOECHO) $(DOCKER_COMPOSE) logs -f -n $(TAIL_LAST_N_LINES) worker

status: # Returns the status of containers
	$(NOECHO) $(DOCKER_COMPOSE) ps -a