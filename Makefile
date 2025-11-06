.PHONY: squash


squash: HEAD := $(shell git rev-parse HEAD)
squash: CURRENT_BRANCH := $(shell git branch --show-current)
squash: MERGE_BASE := $(shell git merge-base origin/main $(CURRENT_BRANCH))

default: test

# run tests with coverage
coverage:
	go test -tags=sqlite -coverprofile=c.out ./...
	go tool cover -html=cover.out

install-gotestsum:
	(cd /tmp && go install gotest.tools/gotestsum@latest)

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	@go test -v -race -short ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -race -run Integration ./...

clean:
	rm -fR bin
	rm -f cover.* junit.xml *.out

squash:
	@git diff --quiet || (echo "you have untracked changes, stopping" && exit 1)
	git branch safe/$(CURRENT_BRANCH)
	git reset $(MERGE_BASE)
	git stash
	git reset --hard origin/main
	git stash apply
	git add .
	git commit -m "Squash changes from $(CURRENT_BRANCH) into main"

docker-compose:
	docker compose up -d && docker compose logs --tail 10 -f

clean-docker-compose:
	docker compose down -v && docker compose rm -f -v

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix
