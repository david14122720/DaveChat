# Delta for User Auth

## ADDED Requirements

### Requirement: Refresh Token Rotation

The system MUST expose `POST /api/auth/refresh` accepting `{refresh_token}` and returning a new access token + new refresh token. The old refresh token MUST be invalidated (rotation).

#### Scenario: Valid refresh token returns new pair

- GIVEN Alice has a valid refresh token stored in `refresh_tokens`
- WHEN Alice calls `POST /api/auth/refresh` with `{refresh_token: "rt_abc123"}`
- THEN response is HTTP 200 with `{token: <new_jwt>, refresh_token: <new_rt>}`
- AND the old `refresh_tokens` row is deleted
- AND a new row with the new token hash is inserted
- AND the new JWT contains the same `sub` claim

#### Scenario: Reused refresh token is rejected

- GIVEN Alice's refresh token `rt_abc123` was already rotated (row deleted)
- WHEN Alice calls refresh again with the same token
- THEN response is HTTP 401 with error code `INVALID_REFRESH_TOKEN`
- AND ALL active refresh tokens for Alice are revoked (compromised token detection)

#### Scenario: Expired refresh token is rejected

- GIVEN Alice's refresh token has `expires_at` in the past
- WHEN Alice calls refresh with that token
- THEN response is HTTP 401 with error code `EXPIRED_REFRESH_TOKEN`

### Requirement: Refresh Token Issued at Login

The system SHOULD issue a refresh token alongside the access token in `POST /api/auth/login` and `POST /api/auth/register`.

#### Scenario: Login returns refresh token

- GIVEN Alice logs in with valid credentials
- WHEN `POST /api/auth/login` succeeds
- THEN response includes `refresh_token` alongside `token` and `user`
- AND the `refresh_token` is persisted as a SHA-256 hash in `refresh_tokens` with 30-day expiry

## MODIFIED Requirements

### Requirement: JWT Middleware

(Previously: Middleware validates JWT and sets user_id in context via `context.WithValue`)

The system MUST require a valid JWT `Authorization: Bearer <token>` header on protected routes. After migration to Echo v4, the middleware MUST use `c.Set("user_id", id)` instead of `context.WithValue`.

#### Scenario: Valid token allows access

- GIVEN a valid JWT for user Alice
- WHEN `GET /api/profiles` with `Authorization: Bearer <token>`
- THEN response is HTTP 200
- AND `c.Get("user_id")` returns Alice's ID

#### Scenario: Missing token returns 401

- GIVEN no authorization header
- WHEN a protected route is called
- THEN response is HTTP 401 with JSON error body
