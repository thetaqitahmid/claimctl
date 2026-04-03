# claimctl Documentation

Welcome to the official documentation for **claimctl**. This documentation
covers everything from setting up the system to using its core features.

## Table of Contents

- **[Deployment & Setup](DEPLOYMENT.md)**
  - Learn how to deploy claimctl using Docker and configure the
    environment.

- **[Authentication](AUTHENTICATION.md)**
  - Local login, LDAP integration, and user sessions.

- **[Resource Management](RESOURCES.md)**
  - Creating, browsing, and managing resources and their history.

- **[Spaces](ADMINISTRATION.md#space-management)**
  - Organizing resources into logical spaces (e.g., "Lab A", "Conference Room").

- **[Reservations & Queue](RESERVATIONS.md)**
  - How to book resources, join waitlists, and manage your reservations.

- **[Administration](ADMINISTRATION.md)**
  - User management, system settings, and administrative controls.

- **[Webhooks & Secrets](WEBHOOKS.md)**
  - Automating workflows with webhooks and managing secure secrets.

- **[CLI Reference](CLI.md)**
  - Command-line interface for advanced users and automation.

---

## Quick Start Summary

1. **Deploy**: Follow the [Deployment Guide](DEPLOYMENT.md) to get the stack
   running.
2. **Login**: The first user created via seed or LDAP is typically an Admin.
3. **Setup**: Create [Spaces](ADMINISTRATION.md#space-management) and then add
   [Resources](RESOURCES.md) to them.
4. **Invite**: Users can now log in and start making
   [Reservations](RESERVATIONS.md).
