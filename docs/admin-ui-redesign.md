# Administration UI redesign

## Navigation

Administration is a top level destination for administrators. Account actions
remain in the account menu. The administration sidebar becomes a collapsible
navigation panel on smaller screens.

| Destination | Route |
| :--- | :--- |
| Users | `/admin/users` |
| Groups | `/admin/groups` |
| Spaces | `/admin/spaces` |
| Space access | `/admin/spaces/:spaceId` |
| Webhooks | `/admin/webhooks` |
| Secrets | `/admin/secrets` |
| Authentication | `/admin/settings/authentication` |
| Notifications | `/admin/settings/notifications` |

`/admin` redirects to Users. `/admin/settings` redirects to Authentication.
Existing `/secrets` and `/webhooks` links redirect to their new destinations.
Every administration route remains behind the administrator role guard.
Backend authorization remains authoritative.

## Settings behavior

1. Authentication and notification settings use grouped forms with friendly
   labels and suitable input types.
2. Only changed fields are submitted. Secrets are never initialized with the
   masked API value. Replacing a secret requires an explicit action and a
   nonempty replacement. Successful replacements are cleared from the form.
3. Saves use the existing individual setting API. They are not atomic.
   Successful fields become the saved baseline. Failed fields remain editable,
   display errors, and are the only fields retried on the next save.
4. Discard restores the most recent saved baseline. Route changes, including
   browser back navigation, prompt before discarding unsaved changes. Browser
   unload uses the browser's native warning.
5. During saving, application navigation asks the operator to wait. Leaving
   the browser cannot cancel a request already sent to the backend.
6. Background refresh errors preserve the loaded form and its edits.

Advanced authentication options retain access to signing key settings. The
interface warns that key updates are individual operations rather than atomic
key pair rotation. Database values may have environment fallbacks; this UI does
not introduce a secret deletion operation with ambiguous fallback behavior.

## Access and safety

1. Users, groups, and spaces have focused pages with search and pagination.
   Pagination is client side because the current APIs return complete lists.
2. Space access is a dedicated page, with separate explanations for direct
   user grants, group membership access, and the public Default Space.
3. Grant lists are not presented as an exhaustive effective access report.
   Administrators retain access, and multiple grants may cover the same person.
4. Shared confirmation dialogs support keyboard focus management, pending
   states, inline failures, and retry without closing on failure.
5. Application backup and restore controls remain removed. Database recovery
   follows the [PostgreSQL recovery guide](BACKUP_RESTORE.md).
6. Resource reservation actions have visible labels. Uncommon administrator
   operations are grouped in a separate keyboard accessible action menu.

## Validation

Run `make test` for backend tests, frontend lint, frontend tests, the production
build, and CLI tests. Frontend tests cover navigation, search and pagination,
settings saves and partial failures, unsaved changes, secret replacement,
permission grants, creation errors, confirmations, and absence of the removed
application backup interface.

The frontend test runner uses at most four workers to avoid interaction test
timeouts caused by launching too many DOM environments concurrently.

Desktop and mobile Chromium smoke checks were also performed with mocked API
responses. These checks do not establish connectivity to real identity or mail
providers or exercise PostgreSQL recovery against a live database.

## Backend capabilities not introduced

This redesign does not add test email delivery, identity provider connection
tests, atomic settings updates, aggregate member/resource counts, or a complete
effective access endpoint. Those require separate API work. The interface does
not show placeholder controls or fabricated results for these capabilities.
