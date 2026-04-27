# gRPC API Documentation

## Overview

The Real Assessment Platform uses gRPC for synchronous inter-service communication. All protobuf definitions are located in `shared/protobuf/` and are generated for Go and Python.

## Protocol Buffers

All `.proto` files follow the `proto3` syntax and use `buf` for linting, generation, and breaking change detection.

### Generation

```bash
# Generate all service stubs
cd shared/protobuf && buf generate

# Lint proto files
buf lint

# Check for breaking changes
buf breaking --against .git#branch=main
```

---

## Services

### 1. UserService (`user_service.proto`)

Service for user management operations consumed by other services via gRPC.

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `GetUser` | `GetUserRequest` | `User` | Retrieve user by ID |
| `GetUserByEmail` | `GetUserByEmailRequest` | `User` | Retrieve user by email |
| `ValidateToken` | `ValidateTokenRequest` | `ValidateTokenResponse` | Validate JWT and return user context |
| `CheckPermission` | `CheckPermissionRequest` | `CheckPermissionResponse` | Check if user has specific role/permission |
| `ListUsers` | `ListUsersRequest` | `ListUsersResponse` | Paginated user list (admin) |

---

### 2. ResumeService (`resume_service.proto`)

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `GetResume` | `GetResumeRequest` | `Resume` | Retrieve resume by ID |
| `GetResumeByUserId` | `GetResumeByUserIdRequest` | `Resume` | Get latest resume for a user |
| `ListUserResumes` | `ListUserResumesRequest` | `ListUserResumesResponse` | List all resumes for a user |
| `GetParsedData` | `GetParsedDataRequest` | `ParsedResumeData` | Get structured parsing results |

---

### 3. InterviewService (`interview_service.proto`)

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `GetInterview` | `GetInterviewRequest` | `Interview` | Retrieve interview by ID |
| `CreateInterview` | `CreateInterviewRequest` | `Interview` | Create new interview session |
| `ListInterviews` | `ListInterviewsRequest` | `ListInterviewsResponse` | List interviews with filters |
| `GetTranscript` | `GetTranscriptRequest` | `InterviewTranscript` | Retrieve interview transcript |
| `CompleteInterview` | `CompleteInterviewRequest` | `Interview` | Mark interview complete |

---

### 4. AIService (`ai_service.proto`)

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `AnalyzeResume` | `AnalyzeResumeRequest` | `AnalyzeResumeResponse` | Parse and analyze resume text |
| `AnalyzeTranscript` | `AnalyzeTranscriptRequest` | `AnalyzeTranscriptResponse` | Analyze interview transcript |
| `ExtractSkills` | `ExtractSkillsRequest` | `ExtractSkillsResponse` | Extract skills from text |
| `GenerateSummary` | `GenerateSummaryRequest` | `GenerateSummaryResponse` | Generate text summary |
| `MatchCandidate` | `MatchCandidateRequest` | `MatchCandidateResponse` | Match candidate to requirements |

---

### 5. ScoringService (`scoring_service.proto`)

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `GetScores` | `GetScoresRequest` | `CandidateScores` | Get aggregated scores for candidate |
| `SubmitScore` | `SubmitScoreRequest` | `SubmitScoreResponse` | Submit a score component |
| `GetReport` | `GetReportRequest` | `ScoreReport` | Generate comprehensive report |
| `ListRankings` | `ListRankingsRequest` | `ListRankingsResponse` | Get candidate rankings |

---

### 6. AdminService (`admin_service.proto`)

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `GetSystemStats` | `GetSystemStatsRequest` | `SystemStats` | System-wide statistics |
| `GetServiceHealth` | `GetServiceHealthRequest` | `ServiceHealthList` | All services health status |
| `GetAuditLogs` | `GetAuditLogsRequest` | `GetAuditLogsResponse` | Retrieve audit log entries |
| `UpdateConfig` | `UpdateConfigRequest` | `UpdateConfigResponse` | Update system configuration |
| `ManageFeatureFlags` | `ManageFeatureFlagsRequest` | `ManageFeatureFlagsResponse` | Toggle feature flags |

---

## Common Types

### User

```protobuf
message User {
  string id = 1;
  string email = 2;
  string name = 3;
  UserRole role = 4;
  bool email_verified = 5;
  google.protobuf.Timestamp created_at = 6;
}

enum UserRole {
  ROLE_UNSPECIFIED = 0;
  ROLE_CANDIDATE = 1;
  ROLE_INTERVIEWER = 2;
  ROLE_ADMIN = 3;
}
```

### Pagination

```protobuf
message PaginationRequest {
  int32 page = 1;
  int32 limit = 2;
}

message PaginationResponse {
  int32 total = 1;
  int32 page = 2;
  int32 limit = 3;
  int32 total_pages = 4;
}
```

### Error Response

```protobuf
message Error {
  string code = 1;
  string message = 2;
  repeated ErrorDetail details = 3;
}

message ErrorDetail {
  string field = 1;
  string description = 2;
}
```

---

## Error Handling

gRPC services use standard gRPC status codes:

| Status Code | Usage |
|-------------|-------|
| `OK` | Successful operation |
| `INVALID_ARGUMENT` | Request validation failed |
| `NOT_FOUND` | Resource does not exist |
| `ALREADY_EXISTS` | Resource already exists (duplicate) |
| `UNAUTHENTICATED` | Invalid or missing credentials |
| `PERMISSION_DENIED` | User lacks required permissions |
| `INTERNAL` | Unexpected server error |
| `UNAVAILABLE` | Service temporarily unavailable |

---

## Connection Guidelines

### Go Client

```go
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    pb "github.com/assessment/proto/user/v1"
)

conn, err := grpc.Dial(
    "user-service:8081",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewUserServiceClient(conn)
resp, err := client.GetUser(ctx, &pb.GetUserRequest{Id: userId})
```

### Python Client

```python
import grpc
from python_common.protobuf import user_service_pb2, user_service_pb2_grpc

channel = grpc.insecure_channel("user-service:8081")
stub = user_service_pb2_grpc.UserServiceStub(channel)

response = stub.GetUser(
    user_service_pb2.GetUserRequest(id=user_id)
)
```

---

*See also: [OpenAPI Spec](./openapi.yaml) | [Protobuf Files](../../protobuf/)*
