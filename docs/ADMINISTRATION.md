# Administration Guide

This section is for system administrators responsible for managing
claimctl.

## Accessing the Admin Panel

To access the Admin Panel, you must be logged in with an account that has the
`admin` role. Click the **Admin** link in the navigation bar.

## User Management

Admins have full control over user accounts.

- **List Users**: View all registered users.
- **Create User**: Manually add a new user (useful if not using LDAP).
- **Edit User**: Update roles, email, or details.
- **Delete User**: Remove user access.

### Roles

- **User**: Standard access. Can view resources and make reservations.
- **Admin**: Full system access. Can manage users, spaces, resources, and
  settings.

## Space Management

**Spaces** are used to organize resources. A Space might represent a physical
room ("Server Room"), a department ("Marketing"), or a category ("Laptops").

### Managing Spaces

1. Go to **Admin Panel > Spaces**.
2. **Create Space**: Enter a name and description.
3. **Edit/Delete**: modify existing spaces.

> **Note**: Deleting a space that contains resources may have side effects.
> Ensure resources are reassigned or deleted first.

## System Maintenance

### Reservation Oversight

Admins can view **All Reservations** in the system, not just their own.

- Override active reservations.
- Cancel reservations on behalf of other users.
