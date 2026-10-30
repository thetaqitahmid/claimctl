# Release validation

Run these checks from the repository root before releasing reservation,
security, delivery, CLI, or replica changes. Validation does not deploy the
application, apply a Helm release, or contact real resource health targets.

## Standard checks

```sh
make test
(cd backend && go vet ./... && go build ./...)
(cd cli && go vet ./... && go build ./...)
make sqlc
git diff --exit-code -- backend/internal/db
```

`make test` includes Helm validation, backend and CLI tests, frontend lint,
frontend tests, and the production build. Helm must be installed, and fetching
the chart's locked dependencies requires network access to its repository.
Without a database URL, backend integration tests skip; a green standard suite
alone is not sufficient release evidence.

Run only the chart checks with `make helm_lint`. This registers the dependency
repository, builds from `Chart.lock`, and runs both lint and template checks
using `charts/claimctl/ci/lint-values.yaml`. The Helm workflow uses the same
synthetic values. They are validation fixtures, not deployment defaults:
actual releases must configure their own trusted frontend and ingress peers.

## Disposable database integration tests

The test helper requires a PostgreSQL cluster owned by the developer at
`127.0.0.1:55432` or `127.0.0.1:55433`. It creates UUID-named databases,
applies all migrations, and drops those databases after each test. The login
must have permission to create databases. Never supply a production URL.

One option is a temporary local Docker container with no persistent volume:

```sh
docker --context default run --rm -d --name claimctl-validation-db \
  -e POSTGRES_USER=claimctl_test \
  -e POSTGRES_PASSWORD=test_only \
  -p 127.0.0.1:55433:5432 postgres:18.1-alpine
```

Wait for PostgreSQL to be ready:

```sh
docker --context default exec claimctl-validation-db \
  pg_isready -U claimctl_test
```

Set the fixture connection URL and ensure the proxy image is local:

```sh
export SECURITY_TEST_DATABASE_URL=\
'postgres://claimctl_test:test_only@127.0.0.1:55433/postgres?sslmode=disable'
docker --context default pull nginx:alpine
make test_integration
```

This target fails immediately if the database URL or local proxy image is
missing. It runs both Go modules with the race detector and disables cached
results. Integration coverage includes:

1. Current-account authorization, token revocation, scoped reads, and CSRF.
2. Atomic claims, idempotency, quotas, queue contention, bundles, and leases.
3. Expired claims releasing quota and same-owner reacquisition before workers
   run, including lease, scheduled hold, and hard deadlines.
4. Transactional delivery, retries, fencing, timeouts, restore, and recovery.
5. Two-server shared state, realtime visibility, proxy origin handling, and
   unbuffered SSE.
6. Populated upgrade and downgrade through migrations 21 to 25, preserving
   business data and not replaying historical delivery intent.

The encryption subprocess helper intentionally skips when invoked directly;
its parent test invokes it with the required helper environment.

Remove only the disposable container you created:

```sh
docker --context default stop claimctl-validation-db
unset SECURITY_TEST_DATABASE_URL
```

## Deployment and terminal checks

```sh
make test_deployment
make test_cli_terminal
(cd cli && go test -shuffle=on -count=10 ./cmd ./pkg/api)
```

Deployment checks require Python with PyYAML, Helm, Docker Compose, and local
`nginx:alpine` and `postgres:18.1-alpine` images. Container-based tests
explicitly use Docker's local `default` context; the selected SSH/remote
context is not used. Bind mounts and loopback proxy tests require the daemon
to share the host filesystem and networking. The Helm fixture CIDR is
synthetic, not a production configuration recommendation.

Terminal checks require Unix PTY support. They start a temporary CLI binary
and a loopback HTTP fixture, checking foreground input, Ctrl+C, cleanup, and
exit status without using real reservations.

## Browser workflow acceptance

The resource workflow has component regression coverage for:

1. Server-policy defaults, explicit indefinite opt-in, and bounded positive
   custom durations.
2. Duplicate-submit prevention and no dismissal during submission.
3. Keeping failed submissions open with actionable feedback and preserving
   idempotency keys when the same request is retried.
4. Retaining table filters and data during refresh, mutation, and fetch errors.
5. A mobile space selector inside the content column and recovery when a
   stored space selection is no longer available.
6. Named controls, keyboard-selectable durations and filters, Escape/focus
   behavior for dialogs and filters, and translated workflow/status text.

Before deployment, also smoke-test the running application in the browsers
and viewport sizes you support. Automated component tests do not substitute
for visual inspection, assistive-technology testing, or production acceptance.
