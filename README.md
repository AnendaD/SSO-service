# SSO Service

## Features

- **JWT Authentication** - Secure token-based authentication with configurable expiration
- **Refresh Token Rotation** - Automatic token refresh with secure rotation mechanism
- **Dual API Support** - Both gRPC and HTTP REST endpoints via proxy server
- **PostgreSQL Storage** - Robust data persistence with migration support
- **Admin Role Management** - Built-in admin user support
- **CORS Support** - Cross-origin resource sharing for web clients
- **Token Validation** - Comprehensive token validation and expiration handling
- **Logout Mechanisms** - Single session and all-sessions logout support

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ├─────────────┐
       │             │
┌──────▼──────┐ ┌───▼──────┐
│ HTTP Proxy  │ │  gRPC    │
│   :8081     │ │  :8080  │
└──────┬──────┘ └───┬──────┘
       │            │
       └────┬───────┘
            │
      ┌─────▼─────┐
      │ Auth      │
      │ Service   │
      └─────┬─────┘
            │
      ┌─────▼─────┐
      │PostgreSQL │
      └───────────┘
```

### Configuration Parameters

- `env` - Environment mode: `local`, `dev`, or `prod`
- `storage_path` - PostgreSQL connection string
- `max_conns` - maximum database connections
- `min_conns` - minimum database connections
- `token_ttl` - Access token lifetime (e.g., `1h`, `15m`)
- `refresh_token_ttl` - Refresh token lifetime (e.g., `720h` = 30 days)
- `grpc.port` - gRPC server port
- `grpc.timeout` - gRPC request timeout
- `http.port` - proxy server port

## Local Setup

```bash
make up
make migrate
```

Run tests:

```bash
make test
```

## Environment Variables

| Variable | Example |
| --- | --- |
| `ENV` | `local` |
| `CONFIG_PATH` | `/app/configs/prod.yaml` |
| `DATABASE_URL` | `postgres://sso:sso@localhost:5432/sso?sslmode=disable` |

### Getting App ID

Before using the service, you need to create an application:

```bash
go run ./internal/lib/app --app_name "MyApp" --db_name "postgres://..."
```

## API Endpoints

### HTTP REST API

#### Register User
```http
POST /api/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword123"
}

Response:
{
  "userId": 1
}
```

#### Login
```http
POST /api/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword123",
  "appId": 1
}

Response:
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "dGVzdC10b2tlbi0xMjM0NTY3ODk..."
}
```

#### Refresh Token
```http
POST /api/refresh
Content-Type: application/json

{
  "refreshToken": "dGVzdC10b2tlbi0xMjM0NTY3ODk..."
}

Response:
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "bmV3LXRva2VuLTk4NzY1NDMyMQ..."
}
```

#### Check Admin Status
```http
POST /api/isAdmin
Content-Type: application/json
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...

{
  "user_id": 1
}

Response:
{
  "isAdmin": false
}
```

#### Logout (Single Session)
```http
POST /api/logout
Content-Type: application/json

{
  "refreshToken": "dGVzdC10b2tlbi0xMjM0NTY3ODk..."
}

Response:
{
  "success": true
}
```

#### Logout All Sessions
```http
POST /api/logoutAll
Content-Type: application/json
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...

{
  "userId": 1,
  "appId": 1
}

Response:
{
  "success": true
}
```

### gRPC API

The service uses Protocol Buffers for gRPC communication. Proto files are available in the [protos repository](https://github.com/AnendaD/protos).

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    pass_hash BYTEA NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE
);
```

### Apps Table
```sql
CREATE TABLE apps (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    secret VARCHAR(255) NOT NULL UNIQUE
);
```

### Refresh Tokens Table
```sql
CREATE TABLE refresh_tokens (
    token VARCHAR(64) PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    app_id INTEGER NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Security Features

### Password Hashing
- Uses bcrypt with default cost factor
- Passwords are never stored in plain text

### JWT Token Security
- HMAC-SHA256 signing algorithm
- Configurable expiration times
- App-specific secrets for signing

### Refresh Token Security
- 32-byte random tokens (base64 encoded)
- Stored securely in database
- Automatic rotation on refresh
- Cascade deletion on user/app removal

### Token Validation
- Signature verification
- Expiration checking
- App-specific secret validation

## Testing

### Run All Tests
```bash
go test ./...
```

### Run Specific Test Suite
```bash
go test ./tests -v
```

## Error Handling

### Common Error Codes

| Error | HTTP Status | Description |
|-------|-------------|-------------|
| `invalid email or password` | 400 | Invalid credentials |
| `user already exists` | 409 | Duplicate registration |
| `token expired` | 401 | Access token expired |
| `invalid or expired refresh token` | 401 | Refresh token invalid/expired |
| `authorization token not provided` | 401 | Missing auth header |

## Monitoring

The service logs all authentication attempts with:
- Operation name
- User identification
- Timestamp
- Success/failure status
