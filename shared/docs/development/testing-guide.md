# Testing Guide

This document covers the testing strategy, patterns, and tooling for the Real Assessment Platform.

---

## Testing Strategy

### Pyramid

```
         /\
        /  \       E2E Tests
       /────\      (few, critical paths)
      /      \
     /────────\    Integration Tests
    /          \   (service boundaries, DB, Kafka)
   /────────────\
  /              \  Unit Tests
 /────────────────\ (most, fast, isolated)
```

| Level | Scope | Speed | Quantity |
|-------|-------|-------|----------|
| **Unit** | Single function/class | < 1ms each | ~80% of tests |
| **Integration** | Service + dependencies | ~100ms each | ~15% of tests |
| **E2E** | Full system flow | ~seconds each | ~5% of tests |

### Targets

- **Line coverage:** > 80% for all services
- **Branch coverage:** > 70% for critical paths
- **All tests pass** in CI before merge
- **E2E tests pass** against staging before production deploy

---

## Unit Testing

### Go

**Framework:** Standard `testing` package + `testify` for assertions

```go
package user_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/assessment/user-service/internal/model"
    "github.com/assessment/user-service/internal/service"
)

func TestCreateUser(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name        string
        email       string
        password    string
        expectError bool
    }{
        {"valid input", "user@example.com", "SecureP@ss1!", false},
        {"invalid email", "not-an-email", "SecureP@ss1!", true},
        {"short password", "user@example.com", "short", true},
        {"empty email", "", "SecureP@ss1!", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            repo := &mockUserRepository{}
            svc := service.NewUserService(repo)

            user, err := svc.CreateUser(context.Background(), &service.CreateUserInput{
                Email:    tt.email,
                Password: tt.password,
            })

            if tt.expectError {
                assert.Error(t, err)
                assert.Nil(t, user)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.email, user.Email)
                assert.NotEmpty(t, user.ID)
            }
        })
    }
}
```

**Mocks:** Use interfaces and manual mocks or `gomock`:

```go
// Define interface in repository layer
type UserRepository interface {
    FindByID(ctx context.Context, id string) (*model.User, error)
    Create(ctx context.Context, user *model.User) error
}

// Mock in tests
type mockUserRepository struct {
    users map[string]*model.User
}

func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
    user, ok := m.users[id]
    if !ok {
        return nil, service.ErrUserNotFound
    }
    return user, nil
}
```

### Python

**Framework:** `pytest` + `pytest-asyncio` + `pytest-mock`

```python
import pytest
from app.services.user_service import UserService
from app.models.user import User


@pytest.mark.asyncio
async def test_create_user_success(mocker):
    # Arrange
    mock_repo = mocker.AsyncMock()
    mock_repo.create.return_value = User(id="user-1", email="test@example.com", name="Test")

    service = UserService(mock_repo)

    # Act
    result = await service.create_user(
        email="test@example.com",
        password="SecureP@ss1!",
        name="Test",
    )

    # Assert
    assert result.email == "test@example.com"
    assert result.id == "user-1"
    mock_repo.create.assert_called_once()


@pytest.mark.asyncio
async def test_create_user_duplicate_email(mocker):
    mock_repo = mocker.AsyncMock()
    mock_repo.create.side_effect = DuplicateEmailError("test@example.com")

    service = UserService(mock_repo)

    with pytest.raises(DuplicateEmailError):
        await service.create_user(
            email="test@example.com",
            password="SecureP@ss1!",
            name="Test",
        )
```

**Test Structure:** Arrange-Act-Assert (AAA) pattern.

---

## Integration Testing

### Go

Use `testcontainers-go` for real dependencies:

```go
func TestInterviewRepository(t *testing.T) {
    ctx := context.Background()

    // Start PostgreSQL container
    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
    )
    require.NoError(t, err)
    t.Cleanup(func() { pgContainer.Terminate(ctx) })

    // Get connection string
    connStr, err := pgContainer.ConnectionString(ctx)
    require.NoError(t, err)

    // Run migrations
    err = runMigrations(connStr)
    require.NoError(t, err)

    // Test
    repo := repository.NewInterviewRepository(connStr)

    interview, err := repo.Create(ctx, &model.Interview{
        CandidateID:  "cand-1",
        InterviewerID: "int-1",
        Status:       "scheduled",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, interview.ID)

    // Verify read
    found, err := repo.FindByID(ctx, interview.ID)
    require.NoError(t, err)
    assert.Equal(t, interview.ID, found.ID)
}
```

### Python

Use `pytest` with `testcontainers`:

```python
import pytest
from testcontainers.postgres import PostgresContainer
from app.repositories.interview_repo import InterviewRepository


@pytest.fixture(scope="module")
def postgres_container():
    with PostgresContainer("postgres:16-alpine") as postgres:
        yield postgres


@pytest.mark.asyncio
async def test_interview_crud(postgres_container):
    conn_str = postgres_container.get_connection_url()

    repo = InterviewRepository(conn_str)
    await repo.initialize()

    # Create
    interview = await repo.create(
        candidate_id="cand-1",
        interviewer_id="int-1",
        status="scheduled",
    )
    assert interview.id is not None

    # Read
    found = await repo.get_by_id(interview.id)
    assert found.status == "scheduled"

    # Update
    await repo.update(interview.id, status="completed")
    updated = await repo.get_by_id(interview.id)
    assert updated.status == "completed"
```

---

## E2E Testing

E2E tests run against a fully started docker-compose environment.

### Pattern

```python
import httpx
import pytest


@pytest.mark.e2e
class TestInterviewFlow:
    """End-to-end test for the complete interview lifecycle."""

    BASE_URL = "http://localhost:8080/api/v1"

    @pytest.fixture
    def client(self):
        return httpx.Client(base_url=self.BASE_URL)

    @pytest.fixture
    def auth_token(self, client: httpx.Client):
        """Login and return auth token."""
        # Register
        client.post("/auth/register", json={
            "email": "e2e@test.com",
            "password": "E2ETest@123!",
            "name": "E2E Test",
        })
        # Login
        resp = client.post("/auth/login", json={
            "email": "e2e@test.com",
            "password": "E2ETest@123!",
        })
        return resp.json()["access_token"]

    def test_create_and_complete_interview(self, client, auth_token):
        headers = {"Authorization": f"Bearer {auth_token}"}

        # Create interview
        resp = client.post("/interviews", json={
            "candidate_id": "cand-1",
            "interviewer_id": "int-1",
            "scheduled_at": "2026-04-20T14:00:00Z",
        }, headers=headers)
        assert resp.status_code == 201
        interview_id = resp.json()["id"]

        # Complete interview
        resp = client.post(f"/interviews/{interview_id}/complete", json={
            "transcript": "Interviewer: Tell me about yourself.\nCandidate: I am a software engineer...",
            "interviewer_notes": "Good candidate",
            "interviewer_rating": 4,
        }, headers=headers)
        assert resp.status_code == 200
        assert resp.json()["status"] == "completed"
```

### Running E2E Tests

```bash
# Start full environment
make dev-up

# Wait for health
make infra-health

# Run E2E tests
make test-e2e

# Tear down
make dev-down
```

---

## Running Tests

### All Tests

```bash
# Root level
make test

# Per service
cd services/user-service && go test ./... -v -race
cd services/ai-service && pytest
```

### Coverage

```bash
# Go
make test-coverage
# Opens HTML coverage report

# Python
cd services/ai-service && pytest --cov=app --cov-report=html
# Open htmlcov/index.html
```

### Specific Tests

```bash
# Go: run single test
go test -run TestCreateUser -v ./internal/service/

# Go: run tests matching pattern
go test -run "TestUser.*" -v ./...

# Python: run single test
pytest tests/test_user_service.py::test_create_user_success -v

# Python: run tests by marker
pytest -m "not slow"
pytest -m "integration"
```

---

## Test Data

### Factories

Use factory functions for test data generation:

```go
// Go
func testUser(t *testing.T) *model.User {
    t.Helper()
    return &model.User{
        ID:       fmt.Sprintf("user-%s", t.Name()),
        Email:    fmt.Sprintf("test-%s@example.com", t.Name()),
        Name:     "Test User",
        Role:     "candidate",
    }
}
```

```python
# Python
import factory

class UserFactory(factory.Factory):
    class Meta:
        model = User

    id = factory.Sequence(lambda n: f"user-{n}")
    email = factory.Sequence(lambda n: f"user{n}@example.com")
    name = "Test User"
    role = "candidate"
    password_hash = "$2b$12$hashed..."
```

### Fixtures

Store JSON fixtures in `tests/fixtures/`:

```
tests/
├── fixtures/
│   ├── user.json
│   ├── interview-completed.json
│   └── ai-analysis-response.json
└── test_api/
    └── test_interview_flow.py
```

---

## CI Integration

Tests run automatically on every push and PR:

```yaml
# .github/workflows/ci.yml (simplified)
test:
  runs-on: ubuntu-latest
  services:
    postgres:
      image: postgres:16-alpine
      env:
        POSTGRES_PASSWORD: test
      ports:
        - 5432:5432
  steps:
    - uses: actions/checkout@v4
    - name: Run Go tests
      run: make test-go
    - name: Run Python tests
      run: make test-python
    - name: Upload coverage
      run: make coverage-upload
```

---

*See also: [Coding Standards](./coding-standards.md) | [Debugging Guide](./debugging-guide.md)*
