.PHONY: squash


squash: HEAD := $(shell git rev-parse HEAD)
squash: CURRENT_BRANCH := $(shell git branch --show-current)
squash: MERGE_BASE := $(shell git merge-base origin/main $(CURRENT_BRANCH))

# Default target: run tests
default: test

# Run tests with coverage profile and generate HTML coverage report
coverage:
	go test -tags=sqlite -coverprofile=c.out ./...
	go tool cover -html=cover.out

# Install gotestsum tool for improved test output
install-gotestsum:
	(cd /tmp && go install gotest.tools/gotestsum@latest)

# Run unit tests with race detector
test-unit: ## Run unit tests
	@echo "Running unit tests..."
	@go test -v -race -short ./internal/...

# Run integration tests
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -race -run Integration ./...

# Remove generated binaries and test artifacts
clean:
	rm -fR bin
	rm -f cover.* junit.xml *.out

# Squash all commits from current branch into a single commit on main
squash:
	@git diff --quiet || (echo "you have untracked changes, stopping" && exit 1)
	git branch safe/$(CURRENT_BRANCH)
	git reset $(MERGE_BASE)
	git stash
	git reset --hard origin/main
	git stash apply
	git add .
	git commit -m "Squash changes from $(CURRENT_BRANCH) into main"

# Start Docker Compose services and follow logs
docker-compose:
	docker compose up -d && docker compose logs --tail 10 -f

# Stop and remove Docker Compose services and volumes
clean-docker-compose:
	docker compose down -v && docker compose rm -f -v

# Run golangci-lint to check code quality
lint:
	golangci-lint run

# Run golangci-lint with automatic fixes
lint-fix:
	golangci-lint run --fix
