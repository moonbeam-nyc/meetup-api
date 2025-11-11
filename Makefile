.PHONY: dev build clean run stop logs install test

# Default target - build, clean, and run
dev: build clean run

# Install dependencies
install:
	npm install

# Build the Docker image
build:
	docker compose build

# Clean up containers and volumes
clean:
	docker compose down -v

# Run the container (foreground with logs)
run:
	docker compose up

# Stop the container
stop:
	docker compose down

# View logs
logs:
	docker compose logs -f

# Restart the service
restart: stop run

# Run tests (placeholder for future implementation)
test:
	npm test

# Format code
format:
	npm run format

# Lint code
lint:
	npm run lint

# Development with live logs
dev-logs: dev logs
