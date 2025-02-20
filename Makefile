# update frontend

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
	@echo "Building backend..."
	@docker compose -f docker-compose.production.yml build augustin-backend
	@echo "Backend built."

push-backend:
	@echo "Push backend..."
	@docker compose -f docker-compose.production.yml push augustin-backend
	@echo "Backend pushed."

update-db-schema:
	@echo "Updating db ent schema..."
	@cd app && go generate ./ent
	@echo "Db ent schema updated."

build: build-frontend build-backend

push: push-frontend push-backend

