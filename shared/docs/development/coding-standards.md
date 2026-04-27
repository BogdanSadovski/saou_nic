# Coding Standards

This document defines the coding conventions for the Real Assessment Platform. Consistent style improves readability, reduces bugs, and makes code reviews more efficient.

---

## Go Style

### General

- Follow [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- Use `gofmt` for all formatting (enforced by CI)
- Line length: 120 characters maximum
- Package names: short, lowercase, no underscores

### Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Packages | lowercase, no underscores | `user`, `auth`, `db` |
| Types | PascalCase | `User`, `UserService` |
| Interfaces | PascalCase, `-er` suffix | `Reader`, `Validator`, `Authenticator` |
| Variables | camelCase | `userID`, `userName` |
| Constants | PascalCase or camelCase | `MaxRetries`, `defaultTimeout` |
| Functions | PascalCase (exported), camelCase (private) | `GetUser`, `parseToken` |
| Errors | `Err` prefix | `ErrUserNotFound`, `ErrInvalidToken` |

### Project Structure

```
service/
├── cmd/
│   └── server/
│       └── main.go          # Entry point
├── internal/
│   ├── handler/             # HTTP handlers
│   ├── service/             # Business logic
│   ├── repository/          # Database access
│   ├── model/               # Domain models
│   ├── middleware/          # HTTP middleware
│   └── config/              # Configuration loading
├── pkg/                     # Public library code (if any)
├── proto/                   # Protobuf files
├── migrations/              # SQL migrations
├── Dockerfile
├── go.mod
└── Makefile
```

### Error Handling

```go
// Define sentinel errors
var (
    ErrUserNotFound = errors.New("user not found")
    ErrInvalidEmail = errors.New("invalid email format")
)

// Return sentinel errors
func GetUser(ctx context.Context, id string) (*User, error) {
    user, err := repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrUserNotFound
        }
        return nil, fmt.Errorf("get user: %w", err)
    }
    return user, nil
}
```

### Logging

Use the shared `logger` package. Include structured context:

```go
import "github.com/assessment/python-common/logger"

log := logger.GetLogger("handler.user")

log.Info("user_created", "user_id", user.ID, "email", user.Email)
log.Error("failed_to_create_user", "error", err, "email", email)
```

### Linting

Run via Makefile or CI:

```bash
# Install linter
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2

# Run
golangci-lint run ./...
```

---

## Python Style

### General

- Follow [PEP 8](https://peps.python.org/pep-0008/) with line length of 120
- Use [Ruff](https://docs.astral.sh/ruff/) for linting and formatting
- Use type annotations everywhere (enforced by mypy)
- Python 3.11+ only

### Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Modules | lowercase, underscores | `user_service`, `auth_middleware` |
| Classes | PascalCase | `UserService`, `JWTValidator` |
| Functions | snake_case | `get_user_by_id`, `validate_token` |
| Variables | snake_case | `user_id`, `access_token` |
| Constants | UPPER_SNAKE_CASE | `MAX_RETRIES`, `DB_POOL_SIZE` |
| Private | leading underscore | `_internal_method`, `_cache_key` |

### Type Annotations

```python
from typing import Optional

async def get_user(
    user_id: str,
    include_profile: bool = False,
) -> Optional[User]:
    """Retrieve user by ID.

    Args:
        user_id: The user's unique identifier.
        include_profile: Whether to include profile data.

    Returns:
        User object or None if not found.
    """
    ...
```

### Project Structure

```
service/
├── app/
│   ├── __init__.py
│   ├── main.py               # Entry point
│   ├── api/                  # Route definitions
│   │   ├── __init__.py
│   │   ├── users.py
│   │   └── auth.py
│   ├── services/             # Business logic
│   │   ├── __init__.py
│   │   └── user_service.py
│   ├── repositories/         # Data access
│   │   ├── __init__.py
│   │   └── user_repo.py
│   ├── models/               # Pydantic models
│   │   ├── __init__.py
│   │   └── user.py
│   └── config.py             # Configuration
├── tests/
│   ├── __init__.py
│   ├── test_api/
│   └── test_services/
├── pyproject.toml
└── requirements.txt
```

### Linting & Formatting

```bash
# Install tools
pip install ruff mypy

# Lint
ruff check .

# Format
ruff format .

# Type checking
mypy app/
```

### Pydantic Models

```python
from pydantic import BaseModel, EmailStr, Field, validator

class CreateUserRequest(BaseModel):
    email: EmailStr
    name: str = Field(..., min_length=1, max_length=100)
    password: str = Field(..., min_length=8)
    role: str = Field(default="candidate")

    @validator("name")
    def name_must_not_be_blank(cls, v: str) -> str:
        if not v.strip():
            raise ValueError("name must not be blank")
        return v.strip()
```

---

## TypeScript Style (Frontend)

### General

- Follow the project ESLint configuration (`.eslintrc.json`)
- Use Prettier for formatting (`.prettierrc`)
- Strict mode enabled in `tsconfig.json`

### Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Components | PascalCase | `UserProfile`, `InterviewCard` |
| Files | PascalCase (components), camelCase (utils) | `UserProfile.tsx`, `api.ts` |
| Hooks | camelCase, `use` prefix | `useAuth`, `useInterview` |
| Types/Interfaces | PascalCase | `User`, `ApiResponse` |
| Variables | camelCase | `userName`, `isLoading` |
| Constants | UPPER_SNAKE_CASE | `API_BASE_URL`, `MAX_FILE_SIZE` |

### Component Structure

```tsx
import React from "react";
import { useAuth } from "@/hooks/useAuth";

interface UserProfileProps {
  userId: string;
}

export const UserProfile: React.FC<UserProfileProps> = ({ userId }) => {
  const { user } = useAuth();

  return (
    <div className="user-profile">
      <h1>{user?.name}</h1>
    </div>
  );
};
```

---

## General Conventions

### Git Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

feat(user): add email verification endpoint
fix(ai): handle OpenAI rate limiting
docs(api): update scoring service docs
chore(deps): update Go to 1.21
refactor(resume): extract parsing logic into service
test(interview): add integration tests for completion
```

**Types:** `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `style`, `perf`, `ci`, `revert`

### API Response Format

All API responses follow a consistent format:

```json
{
  "data": { ... },
  "error": null,
  "meta": {
    "request_id": "req-abc-123",
    "timestamp": "2026-04-07T10:00:00Z"
  }
}
```

Error responses:

```json
{
  "data": null,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request body",
    "details": [
      { "field": "email", "message": "invalid email format" }
    ]
  }
}
```

### Configuration

- Use environment variables for secrets (never hardcode)
- Use YAML config files for non-secret settings
- Validate config on startup, fail fast if invalid
- Provide sensible defaults for all optional settings

---

*See also: [Getting Started](./getting-started.md) | [Git Workflow](./git-workflow.md)*
