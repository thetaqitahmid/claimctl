.PHONY: backend_up backend_down frontend_up dev_up dev_down db_up db_down test

db_up:
	@docker compose -f tests/db/postgres-compose.yaml up -d

db_down:
	@docker compose -f tests/db/postgres-compose.yaml down

backend_up:
	@make db_up
	@sleep 5
	@bash -c "source ./backend/.env && cd backend && migrate --path migrations -database \"postgresql://devuser:devpass@127.0.0.1:5432/devdb?sslmode=disable\" up"
	@bash -c "psql postgresql://devuser:devpass@127.0.0.1:5432/devdb -f ./tests/db-mock-data/seed.sql"
	@bash -c "set -a && source ./backend/.env && set +a && cd backend && go build -o main cmd/main.go && ./main"

backend_down:
	@make db_down
	@rm -f backend/main

frontend_up:
	@cd frontend && npm run dev

dev_up:
	@make -j2 backend_up frontend_up

dev_down: backend_down

test:
	@cd backend && go test ./...
	@cd frontend && npm run lint && npm run test && npm run build
	@cd cli && go test ./...

migrate_up:
	@bash -c "source ./backend/.env && cd backend && migrate --path migrations -database \"postgresql://devuser:devpass@127.0.0.1:5432/devdb?sslmode=disable\" up"

migrate_force:
	@bash -c "source ./backend/.env && cd backend && migrate --path migrations -database \"postgresql://devuser:devpass@127.0.0.1:5432/devdb?sslmode=disable\" force $(VERSION)"

sqlc:
	@cd backend && sqlc generate
