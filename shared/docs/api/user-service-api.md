# User Service API

## Overview

The User Service manages user accounts, authentication, and role-based access control. It is implemented in Go and exposes a REST API through the API Gateway.

**Base URL:** `/api/v1`
**Tech Stack:** Go, PostgreSQL, Redis

---

## Endpoints

### Authentication

#### POST /auth/register

Register a new user account.

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "SecureP@ssw0rd!",
  "name": "John Doe",
  "role": "candidate"
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `email` | string | Yes | Valid email, unique |
| `password` | string | Yes | Min 8 chars, uppercase, digit, special char |
| `name` | string | Yes | 1-100 characters |
| `role` | string | No | `candidate`, `interviewer`, `admin` (default: `candidate`) |

**Response (201 Created):**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJl...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "name": "John Doe",
    "role": "candidate",
    "email_verified": false,
    "created_at": "2026-04-07T10:00:00Z"
  }
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid input fields |
| 409 | `EMAIL_EXISTS` | Email already registered |

---

#### POST /auth/login

Authenticate with email and password.

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "SecureP@ssw0rd!"
}
```

**Response (200 OK):** Same as register response.

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 401 | `INVALID_CREDENTIALS` | Wrong email or password |
| 403 | `ACCOUNT_DISABLED` | Account has been deactivated |

---

#### POST /auth/refresh

Get new access token using refresh token.

**Request Body:**

```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJl..."
}
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "bmV3IHJlZnJlc2ggdG9r...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

---

#### POST /auth/logout

Invalidate the current refresh token.

**Headers:** `Authorization: Bearer <token>`

**Response (204 No Content)**

---

### User Management

#### GET /users/me

Get the authenticated user's own profile.

**Headers:** `Authorization: Bearer <token>`

**Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "name": "John Doe",
  "role": "candidate",
  "email_verified": true,
  "avatar_url": "https://cdn.example.com/avatars/550e8400.jpg",
  "bio": "Software engineer with 5 years experience",
  "created_at": "2026-04-07T10:00:00Z",
  "updated_at": "2026-04-07T12:30:00Z"
}
```

---

#### PUT /users/me

Update the authenticated user's profile.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**

```json
{
  "name": "John Smith",
  "avatar_url": "https://cdn.example.com/avatars/new.jpg",
  "bio": "Updated bio text"
}
```

**Response (200 OK):** Updated user object.

---

#### GET /users

List users (admin only).

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page (max 100) |
| `role` | string | - | Filter by role |
| `search` | string | - | Search by name or email |

**Response (200 OK):**

```json
{
  "items": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "name": "John Doe",
      "role": "candidate",
      "email_verified": true,
      "created_at": "2026-04-07T10:00:00Z"
    }
  ],
  "total": 150,
  "page": 1,
  "limit": 20
}
```

---

#### GET /users/{user_id}

Get a specific user by ID.

**Headers:** `Authorization: Bearer <token>`

**Response (200 OK):** User object.

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 404 | `USER_NOT_FOUND` | User does not exist |

---

## gRPC Interface

Other services consume the User Service via gRPC:

| Method | Description |
|--------|-------------|
| `GetUser` | Retrieve user by ID |
| `GetUserByEmail` | Retrieve user by email |
| `ValidateToken` | Validate JWT and return user context |
| `CheckPermission` | Check if user has a specific role |

See [gRPC API Documentation](./grpc-api.md) for protobuf definitions.

---

*See also: [OpenAPI Spec](./openapi.yaml) | [gRPC API](./grpc-api.md)*
