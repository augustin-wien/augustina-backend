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

