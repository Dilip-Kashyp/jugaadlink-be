APP_NAME=url-shortener

dev:
	go run ./cmd/server/main.go
	

prod:
	docker compose up --build

build:
	docker build -t $(APP_NAME) .

up:
	docker compose up

down:
	docker compose down

clean:
	docker compose down -v
	docker image prune -f

logs:
	docker compose logs -f

db:
	docker exec -it postgres_db psql -U $(DB_USER) -d $(DB_NAME)

# Production deploy (used by CI/CD)
deploy:
	git pull origin main
	docker compose down
	docker compose up -d --build
	docker image prune -f
