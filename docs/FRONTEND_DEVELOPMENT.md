# claimctl Frontend

React-based frontend for the claimctl resource management system.

## Tech Stack

- **React 18** with TypeScript
- **Vite** for development and building
- **Redux Toolkit** for state management
- **React Router** for navigation
- **Tailwind CSS** for styling
- **Vitest** for unit testing

## Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Run linting
npm run lint
```

## Testing

The project uses **Vitest** with **React Testing Library** for unit testing.

### Running Tests

```bash
# Run tests (single run)
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage report
npm run test:coverage
```

### Test Structure

Tests are co-located with the source files using the `.test.tsx` or `.test.ts`
suffix:

```
src/
 components/
 Toast.tsx
 Toast.test.tsx # Component test
 hooks/
 useNotification.ts
 useNotification.test.tsx # Hook test
 store/slices/
 authSlice.ts
 authSlice.test.ts # Redux slice test
 test/
 setup.ts # Test setup (jest-dom, mocks)
 utils.tsx # Custom render with providers
```

### Test Utilities

Use the custom `renderWithProviders` function from `src/test/utils.tsx` for
components that need Redux or Router context:

```tsx
import { renderWithProviders, screen } from "../test/utils";
import MyComponent from "./MyComponent";

it("renders with Redux state", () => {
  renderWithProviders(<MyComponent />, {
    preloadedState: {
      authSlice: { user: "testuser", email: null, admin: false, role: null },
    },
  });

  expect(screen.getByText("testuser")).toBeInTheDocument();
});
```

### Writing Tests

**Component tests:**

```tsx
import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import MyComponent from "./MyComponent";

describe("MyComponent", () => {
  it("renders correctly", () => {
    render(<MyComponent title="Hello" />);
    expect(screen.getByText("Hello")).toBeInTheDocument();
  });
});
```

**Redux slice tests:**

```tsx
import { describe, it, expect } from "vitest";
import reducer, { someAction } from "./mySlice";

describe("mySlice", () => {
  it("handles someAction", () => {
    const result = reducer(initialState, someAction(payload));
    expect(result.value).toBe(expected);
  });
});
```

**Hook tests:**

```tsx
import { describe, it, expect } from "vitest";
import { renderHook } from "@testing-library/react";
import { useMyHook } from "./useMyHook";

describe("useMyHook", () => {
  it("returns expected value", () => {
    const { result } = renderHook(() => useMyHook());
    expect(result.current.value).toBe(expected);
  });
});
```

### Current Test Coverage

| Category      | Files  | Tests  |
| ------------- | ------ | ------ |
| Redux Slices  | 2      | 8      |
| Hooks         | 2      | 7      |
| UI Components | 4      | 18     |
| Components    | 6      | 44     |
| **Total**     | **14** | **77** |
