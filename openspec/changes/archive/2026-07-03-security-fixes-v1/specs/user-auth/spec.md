# Delta for user-auth

## ADDED Requirements

### Requirement: Logger Query String Omission

The Echo Logger MUST omit query strings from logged request URIs.

#### Scenario: Request URI logged without query strings

- GIVEN a request to `/api/auth/login?token=secret`
- WHEN Echo processes and logs the request
- THEN the log entry contains `/api/auth/login`
- AND the log entry MUST NOT contain the query string `token=secret`

## MODIFIED Requirements

### Requirement: Register

The system MUST expose `POST /api/auth/register` accepting email, password, and username. On registration conflict — duplicate email or duplicate username — the system MUST return a single generic error message that does not distinguish between conflict types.
(Previously: returned distinct error messages for email vs username conflicts)

#### Scenario: Valid registration creates user and returns JWT

- GIVEN no user exists with email `a@b.com`
- WHEN `POST /api/auth/register` with `{email: "a@b.com", password: "Secret1!", username: "Alice"}`
- THEN response is HTTP 201 with `{token, user: {id, email, username}}`
- AND the password is stored as bcrypt hash
- AND a `users` row is created

#### Scenario: Duplicate email or username returns generic error

- GIVEN a user exists with email `a@b.com` (or username `Alice`)
- WHEN `POST /api/auth/register` with the same email or username
- THEN response is HTTP 409
- AND response body is `{"error":"Registration failed. Email or username may already be in use."}`

### Requirement: JWT Middleware

The system MUST require a valid JWT via `Authorization: Bearer <token>` header on protected routes. The middleware MUST reject any JWT supplied via `?token=` query parameter and return HTTP 401.
(Previously: accepted JWT via either Authorization header or `?token=` query parameter)

#### Scenario: Valid token allows access

- GIVEN a valid JWT for user Alice
- WHEN `GET /api/profiles/me` with `Authorization: Bearer <token>`
- THEN response is HTTP 200

#### Scenario: Missing token returns 401

- GIVEN no authorization header
- WHEN `GET /api/profiles/me` is called
- THEN response is HTTP 401

#### Scenario: Expired token returns 401

- GIVEN a JWT with past `exp`
- WHEN a protected route is called
- THEN response is HTTP 401 with "token expired"

#### Scenario: Token via query param returns 401

- GIVEN a request with `?token=<valid_jwt>` and no `Authorization` header
- WHEN a protected route is called
- THEN response is HTTP 401
- AND response body contains "Authorization header required"
