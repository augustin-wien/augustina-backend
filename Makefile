VERSION_FILE=app/config/config.go

update-frontend:
	@echo "Updating frontend..."
	@cd docker/augustin-frontend && git pull
	@echo "Frontend updated."

build-frontend:
	@make update-frontend
	@echo "Building frontend..."
	@docker compose -f docker-compose.production.yml build augustin-frontend
	@echo "Frontend built."

push-frontend:
	@echo "Push frontend..."
	@docker compose -f docker-compose.production.yml push augustin-frontend
	@echo "Frontend pushed."

build-backend:
	@make update-version
	@echo "Building backend..."
	@export GIT_COMMIT=$(git rev-parse --short HEAD) && docker compose -f docker-compose.production.yml build augustin-backend
	@echo "Backend built."

push-backend:
	@echo "Push backend..."
	@docker compose -f docker-compose.production.yml push augustin-backend
	@echo "Backend pushed."

update-db-schema:
	@echo "Updating db ent schema..."
	@cd app && go generate ./ent
	@echo "Db ent schema updated."

update-version:
	@echo "Updating version in $(VERSION_FILE)..."
	@python3 scripts/update_version.py

update-docker-containers:
	@echo "Updating docker containers..."
	@docker pull alpine:latest
	@docker pull golang:latest
	@docker pull node:24
	@echo "Docker containers updated."

build: build-frontend build-backend

push: push-frontend push-backend

.PHONY: ci-local

ci-local:
	@# require either a local ./tools/act or a system 'act'
	@if [ -x ./tools/act ] || command -v act >/dev/null 2>&1; then \
		echo "Running GitHub Actions job 'Gotest' locally with act..."; \
	else \
		echo "Please install 'act' (https://github.com/nektos/act) or run 'make install-act'"; exit 1; \
	fi
	@# If .env exists, pass it to act so services get the same environment
	@if [ -f ".env" ]; then \
		ENVFILE="--env-file .env"; \
	else \
		ENVFILE=""; \
	fi; \
	# Use a known act runner image for ubuntu-latest; adjust if needed
	# Prefer local tools/act if present
	if [ -x ./tools/act ]; then \
		ACT_BIN=./tools/act; \
	else \
		ACT_BIN=act; \
	fi; \
	$$ACT_BIN -j Gotest $$ENVFILE -P ubuntu-latest=ghcr.io/catthehacker/ubuntu:full-22.04

.PHONY: install-act
install-act:
	@echo "Installing act into ./tools"
	@bash scripts/install_act.sh

