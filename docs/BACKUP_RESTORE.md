# PostgreSQL backup and recovery

claimctl uses PostgreSQL native backups. There is no application JSON backup
format, backup API, restore API, browser upload, or claimctl backup command.
Use `pg_dump` and `pg_restore`, or your database provider's supported backup
and recovery service. A raw copy of a running PostgreSQL data directory is
not a supported backup procedure.

A fresh claimctl deployment can use a restored database directly. No
application import or reseeding is required. Preserve the original encryption
key, deployment configuration, and a compatible application version.

## Recovery material

Retain these together as a recovery set, with appropriate access controls:

1. A complete database backup, including both the `claimctl` schema and the
   `public` schema. `public.app_settings` contains application settings and
   signing keys; `public.schema_migrations` records the schema version.
2. The exact original `APP_ENCRYPTION_KEY`, a base64 encoded 32 byte key.
   If development used `keys/app.key`, preserve that file securely or supply
   its value as `APP_ENCRYPTION_KEY` on recovery. Never generate a replacement
   key for an existing database.
3. Compose or Kubernetes manifests, Helm values, external Secrets, database
   connection settings, TLS certificates, OIDC credentials, and other external
   configuration. Preserve Secret values, not just their resource names.
4. The application image digest, PostgreSQL major version, backup timestamp,
   and a checksum of the completed archive.

The database contains users, password hashes, resources, spaces, groups,
permissions, reservations, histories, API token hashes, encrypted secrets,
webhooks, notification preferences, settings, and audit logs. Dumps are
sensitive even though some fields are encrypted. Store encrypted copies
outside the database host, with restricted permissions and tested retention.
The database does not contain your deployment manifests or encryption key.

Kubernetes generated encryption Secrets must be preserved explicitly. A new
Helm release that generates a new key cannot decrypt the recovered database.
Existing API token plaintext cannot be recovered from its stored hash.

## Create a backup

Use PostgreSQL client tools matching the source server's major version.
Restore first to the same PostgreSQL major version and application image;
perform upgrades separately. Never restore to an older PostgreSQL version.
Provision the database role and required database privileges separately:
`pg_dump` does not back up cluster roles.

The examples use libpq service definitions in `~/.pg_service.conf`, with
passwords supplied through a restricted `~/.pgpass` or `PGPASSFILE`. Configure
TLS verification for remote databases. Do not commit credentials or put
passwords in shell history. The service names are examples, not claimctl
configuration keys.

Configure `claimctl_source` to connect to the source database, then run:

```bash
set -eu
umask 077
archive="claimctl-$(date -u +%Y%m%dT%H%M%SZ).dump"
pg_dump --dbname='service=claimctl_source' \
  --format=custom --file="$archive.partial"
pg_restore --list "$archive.partial" > "$archive.contents"
mv "$archive.partial" "$archive"
sha256sum "$archive" > "$archive.sha256"
```

Dump the entire database, without schema or table filters. `pg_dump` takes a
consistent snapshot while the application is running. It does not stop
external webhook requests or include changes committed after its snapshot.
The archive listing and checksum are useful checks, but neither substitutes
for a successful isolated restore and application validation.

Automate this procedure using your scheduler or managed database service.
Alert on failures, missing backups, and failed recovery drills. Choose backup
frequency and retention to meet your recovery point and recovery time goals.
For point in time recovery, use managed recovery or physical backups with WAL
archiving instead of relying only on periodic logical dumps.

## Restore before starting claimctl

1. Stop all claimctl replicas and workers, including any old deployment that
   could write to the recovered database or send webhook requests. Keep
   application traffic blocked. Do not start the replacement deployment yet.
2. Verify the archive checksum and recovery material. Restore only trusted
   archives: database dumps can execute SQL during restoration.
3. Provision a fresh, empty database from `template0`, owned by the intended
   application database role. Do not run claimctl migrations or seed scripts
   before restoring. Keep the old database available for rollback.
4. Restore the full archive, then run the runtime cleanup below.
5. Configure a fresh deployment with the restored database, original
   encryption key, same manifests and Secret values, and original application
   image. Start one instance, validate it, then resume traffic and replicas.

For example, configure `claimctl_admin` to connect to a maintenance database
on the target server with database creation rights. Configure
`claimctl_recovered` to connect to the new database as the role that should
own the restored objects. Substitute your database and role names:

```bash
createdb --maintenance-db='service=claimctl_admin' \
  --template=template0 --owner=claimctl claimctl_recovered
pg_restore --dbname='service=claimctl_recovered' \
  --no-owner --no-privileges --exit-on-error --single-transaction \
  claimctl-20260101T000000Z.dump
psql 'service=claimctl_recovered' -X --set=ON_ERROR_STOP=1 \
  --file=deployment/postgres-recovery.sql
```

Run from the repository root for the cleanup file path above. The cleanup
script is an explicit offline operator action, not startup behavior. It is
for the current schema; use the script from the backed up application version.
Do not run it on a live database. It intentionally uses no `CASCADE`, so an
unexpected dependency fails instead of silently deleting business data.

The restore flags make the connecting role own the objects and omit old
access grants. Reapply any additional reporting or administrative grants
separately. If ownership and ACL preservation are required, provision the
original roles and use an operator reviewed restore procedure instead.
Single transaction restore can require a higher PostgreSQL lock capacity for
large databases. Do not continue to startup after any restore or cleanup
error. Investigate or recreate the empty target and retry.

## Runtime cleanup policy

A full native dump includes runtime tables. Before any recovered instance
starts, `deployment/postgres-recovery.sql` clears these in one transaction:

| Tables | Reason |
| --- | --- |
| `refresh_tokens` | Do not revive backed up browser refresh sessions |
| `claim_idempotency`, `bundle_idempotency` | Do not replay stale results |
| `webhook_outbox` | Do not resend events already delivered after the backup |
| `worker_leases` | Do not restore old worker ownership |
| `realtime_events` | Clients must refetch recovered application state |
| `shared_storage`, `rate_limits` | Discard old CSRF challenges and counters |

All table names above are in the `claimctl` schema. Business data, audit logs,
health history, settings, and API tokens remain intact. Cleanup intentionally
loses undelivered webhook jobs. Reconcile important external side effects
manually; recovery cannot undo a webhook request already accepted elsewhere.
Do not reuse claim or bundle idempotency keys from before recovery.

Native full restore creates the application triggers after loading data, so
historical reservations do not generate fresh delivery jobs during import.
Do not substitute a data only restore into a running application's schema.

Clearing refresh sessions does not revoke every existing access JWT. Tokens
with the same signing key and account credentials may work until expiry,
normally up to 15 minutes. Retained API tokens also remain usable until expiry
or revocation. Account changes or token revocations made after the backup are
not present in the recovered database: review them before reopening traffic.
For a security incident, follow a separate credential and signing key rotation
procedure; this recovery procedure is not credential revocation.

## Fresh deployment behavior and validation

Point `DATABASE_URL`, or `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`,
`DB_NAME`, and `SSL_MODE`, at the recovered database. Preserve all other
manifest and Secret values, especially `APP_ENCRYPTION_KEY`. For Compose,
restore before starting the backend services. For Kubernetes, keep application
replicas at zero until restoration and cleanup finish. The database itself
must be running. Do not allow a migration init container or Helm hook to
initialize the target before the full restore.

At startup claimctl checks migrations, reuses existing administrator accounts,
and ensures a default space exists. It loads the stored JWT signing key pair
using the supplied encryption key. Partial, corrupt, or undecryptable existing
signing key pairs fail startup rather than being silently replaced. Startup
does not reset existing administrator passwords or import data from an
application file. Environment variables can populate empty database settings
at startup, so preserve the original configuration for consistent recovered
behavior. No runtime tables are automatically cleared on startup: ordinary
restarts must preserve durable jobs and coordination.

Validate before accepting recovery:

1. `/v1/health` and `/v1/ready` return success.
2. An existing administrator can log in with the original credentials.
3. Users, resources, spaces, permissions, and reservation histories match
   expected records. Audit logs and token scope restrictions are preserved.
4. An existing encrypted secret and an encrypted setting decrypt correctly.
   Never print their plaintext in recovery logs.
5. Workers resume without historical delivery replay. New webhook events and
   health checks work under the original network and egress configuration.
6. Restored lease and reservation deadlines remain unchanged. Deadlines that
   elapsed during downtime are reclaimed normally; recovery does not extend
   reservations. Account for worker driven changes during validation.
7. Browser clients authenticate again and refetch current state. Existing
   external integrations use valid API tokens and new idempotency keys.

Keep the original database and recovery material until validation completes.
Rollback requires stopping recovered workers too; switching databases cannot
undo external effects they already produced.

## References

1. [PostgreSQL pg_dump][pg-dump]
2. [PostgreSQL pg_restore][pg-restore]
3. [PostgreSQL backup][pg-backup]

[pg-dump]: https://www.postgresql.org/docs/current/app-pgdump.html
[pg-restore]: https://www.postgresql.org/docs/current/app-pgrestore.html
[pg-backup]: https://www.postgresql.org/docs/current/backup.html
