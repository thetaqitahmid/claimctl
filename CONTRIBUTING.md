# Contributing to claimctl

First off, thank you for considering contributing to claimctl! It is
people like you that make claimctl such a great tool for resource
management.

## Getting Started

1. Fork the repository and create your branch from `main`.
2. Ensure you have the required dependencies: `Go`, `Node.js`, and `Docker`.
3. If you have never built the project before, run `make dev_up` to start the
   full development environment (backend + frontend + db).
4. See `AGENTS.md` and `README.md` for our build and testing scripts.

## Code Style Guidelines

- We use Go for the backend and React/TypeScript for the frontend.
- Please follow the code style guidelines described in `AGENTS.md`.
- Ensure all backend tests pass using `make test`.
- Ensure your code is thoroughly documented, and commit messages are
  descriptive.

## Commit Messages

We follow the Conventional Commits specification for our commit messages. This
format helps create readable history and automated changelogs.

Format: `<type>[optional scope]: <description>`

### Types

- `feat`: A new feature (minor version bump).
- `fix`: A bug fix (patch version bump).
- `docs`: Documentation changes only.
- `style`: Formatting changes (whitespace, missing semicolons, etc.).
- `refactor`: Code changes that neither fix a bug nor add a feature.
- `test`: Adding or correcting tests.
- `chore`: Other changes that don't modify source or test files.

### Scope and Description

- `scope`: (Optional) The specific part of the codebase modified, in
  parentheses (e.g., `(api)`, `(ui)`).
- `description`: A short summary in the imperative, present tense (e.g.,
  "add OAuth login" instead of "added OAuth login"). Do not capitalize the
  first letter or add a period at the end.

### Breaking Changes

Append a `!` after the type/scope for breaking changes (e.g., `refactor!: ...`).

## Pull Requests

1. Create a descriptive pull request.
2. Fill out our provided Pull Request template.
3. Ensure the CI passes (if configured).
4. Wait for a review from the maintainers.

By contributing to this project, you agree that your contributions will be
licensed under its Apache 2.0 License.
