.PHONY: backend_up backend_down frontend_up dev_up dev_down db_up db_down test test_integration test_deployment test_cli_terminal

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

test_integration:
	@test -n "$$SECURITY_TEST_DATABASE_URL" || { echo "Set SECURITY_TEST_DATABASE_URL to a disposable localhost database on port 55432 or 55433" >&2; exit 1; }
	@docker --context default image inspect nginx:alpine >/dev/null
	@cd backend && go test -race -count=1 ./...
	@cd cli && go test -race -count=1 ./...

test_deployment:
	@python3 deployment/helm_replica_test.py
	@python3 deployment/replica_config_test.py
	@sh deployment/postgres-preflight_test.sh
	@helm lint charts/claimctl --set backend.trustedProxyCIDRs=10.244.0.0/16

test_cli_terminal:
	@python3 cli/tests/exec_pty_test.py

migrate_up:
	@bash -c "source ./backend/.env && cd backend && migrate --path migrations -database \"postgresql://devuser:devpass@127.0.0.1:5432/devdb?sslmode=disable\" up"

migrate_force:
	@bash -c "source ./backend/.env && cd backend && migrate --path migrations -database \"postgresql://devuser:devpass@127.0.0.1:5432/devdb?sslmode=disable\" force $(VERSION)"

sqlc:
	@cd backend && sqlc generate
