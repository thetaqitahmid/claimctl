# claimctl Development Guidelines

## Build & Development Commands

### Root Makefile Commands

```bash
# Start full development environment (backend + frontend + db)
make dev_up

# Start/stop backend with database
make backend_up
make backend_down

# Start frontend only
make frontend_up

# Database operations
make db_up      # Start PostgreSQL container
make db_down    # Stop PostgreSQL container
make migrate_up # Run migrations
make sqlc       # Regenerate sqlc code

# Run all tests
make test
```

### Backend (Go)

```bash
cd backend

# Build
go build -o backend cmd/main.go

# Lint
go vet ./...

# Database
sqlc generate                                           # Regenerate DB code
migrate create -ext sql -dir migrations -seq <name>     # New migration
migrate --path migrations -database "<DB_CONNECTION>" up

# Tests
go test ./...                           # Run all tests
go test -run TestUserService_Login ./... # Run specific test by name
go test -v ./...                        # Verbose output
go test -cover ./...                    # With coverage
```

### Frontend (React/TypeScript)

```bash
cd frontend

# Development
npm run dev         # Start Vite dev server
npm run build       # TypeScript compile + Vite build
npm run lint        # ESLint check
npm run preview     # Preview production build
```

## Code Style Guidelines

### Go Backend

**Imports** - Group in order: stdlib, third-party, internal, external project

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/jackc/pgx/v5/pgtype"
    "golang.org/x/crypto/bcrypt"

    "github.com/thetaqitahmid/claimctl/internal/db"
    "github.com/thetaqitahmid/claimctl/internal/services"
    "github.com/thetaqitahmid/claimctl/internal/utils"
)
```

**Package Structure**

- `cmd/` - Application entry points
- `internal/handlers/` - HTTP request handlers (Fiber)
- `internal/services/` - Business logic interfaces and implementations
- `internal/db/` - sqlc-generated database code
- `internal/types/` - Custom types (JSONB, events)
- `internal/utils/` - Utility functions
- `internal/testutils/` - Test helpers and mocks

**Naming Conventions**

- Exported: PascalCase (`UserHandler`, `CreateUser`)
- Unexported: camelCase (`userService`, `generateJWT`)
- Interfaces: Noun with "Service" suffix (`UserService`, `ReservationService`)
- Files: snake_case for handlers (`user_handlers.go`, `reservation_service.go`)
- Test files: `*_test.go` suffix

**Error Handling**

```go
return nil, fmt.Errorf("failed to create user: %w", err)
```

**HTTP Handlers** (Fiber framework)

```go
return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
    "error": "Invalid request body",
    "details": err.Error(),
})
```

**Database Patterns**

- Use sqlc-generated types exclusively (`db.CreateUserParams`,
  `db.claimctlUser`)
- Use `pgtype` types for nullable fields (`pgtype.Text`, `pgtype.Int4`)
- JSONB for flexible arrays (labels, properties, history)
- Use pointers in update request structs for optional fields

**Testing**

- Use `testify` library (assert, require, mock)
- Mock database with `testutils.MockQuerier`
- Table-driven tests with struct pattern
- Test context: `testutils.TestContext()`

### TypeScript Frontend

**Import Order**

```typescript
import { useState, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { createApi } from "@reduxjs/toolkit/query/react";
import { Box, Plus } from "lucide-react";
import { Resource } from "../types";
import { useGetResourcesQuery } from "../store/api/resources";
import ResourceList from "../components/ResourceList";
```

**Component Structure**

```typescript
const ComponentName = () => {
  // hooks first
  const { t } = useTranslation()
  const [state, setState] = useState()

  // derived values
  const computed = useMemo(() => ...)

  // handlers
  const handleClick = () => { ... }

  return ( ... )
}

export default ComponentName
```

**State Management**

- Redux Toolkit with RTK Query for API calls
- Auth context in `store/context/`
- Slices in `store/slices/`
- API slices in `store/api/`

**TypeScript**

- Define interfaces in `types.d.ts`
- Use RTK Query with typed endpoints
- Use `useAppSelector` hook for Redux state

**Styling** (Tailwind CSS)

- Semantic colors: `text-slate-600`, `bg-blue-50`, `border-gray-200`
- Consistent spacing: `p-4`, `m-2`, `gap-4`
- Responsive: `md:`, `lg:` prefixes

### Database Schema

**Table Conventions**

- Plural names: `users`, `resources`, `reservations`
- Snake_case: `created_at`, `user_id`
- Timestamps: `created_at`, `updated_at`, `last_modified`
- Primary key: `id` (SERIAL)
- Use BIGINT epoch timestamps for performance

**Indexes**

- Create on frequently queried fields
- Foreign key columns
- Search fields (email, name)

**Migrations**

- Always create `.up.sql` and `.down.sql` pairs
- Sequential numbering: `000001_`, `000002_`
- Test migrations in both directions

### API Design

**REST Conventions**

- GET /resources - List all
- GET /resources/:id - Get one
- POST /resources - Create
- PATCH /resources/:id - Update
- DELETE /resources/:id - Delete

**Response Format**

```json
{ "error": "message", "details": "..." }
```

**Authentication**

- JWT in HTTP-only cookies
- RS256 signing
- Role-based access (admin, user)

### Security

- Passwords: bcrypt hashing
- JWT: RS256, HTTPOnly cookies
- SQL: Parameterized queries via sqlc
- CORS: Configured in middleware

### Markdown

- Maximum line length 80 characters
- Use backticks for code
- Do not use emojis
