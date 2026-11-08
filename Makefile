.PHONY: backend_up backend_down frontend_up frontend_deps dev_init dev_up dev_down db_up db_down test test_integration test_deployment test_cli_terminal test_dev_start helm_lint seed db_seed

dev_init:
	@bash deployment/dev-env.sh

db_up: dev_init
	@docker compose -f tests/db/postgres-compose.yaml up -d --wait --wait-timeout 90

db_down:
	@docker compose -f tests/db/postgres-compose.yaml down

backend_up: db_up
	@bash -ec 'set -a; source ./backend/.env; set +a; cd backend; go build -o main cmd/main.go && exec ./main'

backend_down:
	@$(MAKE) db_down
	@rm -f backend/main

frontend/node_modules/.package-lock.json: frontend/package-lock.json frontend/package.json
	@cd frontend && npm ci

frontend_deps: frontend/node_modules/.package-lock.json

frontend_up: frontend_deps
	@cd frontend && npm run dev -- --strictPort

dev_up:
	@MAKE="$(MAKE)" bash deployment/dev-up.sh

dev_down: backend_down

seed db_seed: dev_init
	@bash deployment/dev-seed.sh

test_dev_start:
	@python3 deployment/dev-start_test.py

test: helm_lint test_dev_start
	@cd backend && go test ./...
	@cd frontend && npm run lint && npm run test && npm run build
	@cd cli && go test ./...

test_integration:
	@test -n "$$SECURITY_TEST_DATABASE_URL" || { echo "Set SECURITY_TEST_DATABASE_URL to a disposable localhost database on port 55432 or 55433" >&2; exit 1; }
	@docker --context default image inspect nginx:alpine >/dev/null
	@cd backend && go test -race -count=1 ./...
	@cd cli && go test -race -count=1 ./...

helm_lint:
	@helm repo add groundhog2k https://groundhog2k.github.io/helm-charts/
	@helm dependency build charts/claimctl
	@helm lint charts/claimctl -f charts/claimctl/ci/lint-values.yaml
	@helm template claimctl charts/claimctl -f charts/claimctl/ci/lint-values.yaml >/dev/null

test_deployment: helm_lint
	@python3 deployment/helm_replica_test.py
	@python3 deployment/replica_config_test.py
	@sh deployment/postgres-preflight_test.sh

test_cli_terminal:
	@python3 cli/tests/exec_pty_test.py

migrate_up: dev_init
	@bash -ec 'source ./backend/.env; cd backend; migrate --path migrations -database "$$DATABASE_URL" up'

migrate_force: dev_init
	@test -n "$(VERSION)" || { echo "Set VERSION to the required migration version" >&2; exit 1; }
	@bash -ec 'source ./backend/.env; cd backend; migrate --path migrations -database "$$DATABASE_URL" force "$(VERSION)"'

sqlc:
	@cd backend && sqlc generate
