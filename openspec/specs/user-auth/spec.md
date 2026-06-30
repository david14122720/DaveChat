# User Auth Specification

## Purpose

JWT-based registration and login replacing Supabase Auth. The Go server handles password hashing (bcrypt), token issuance, and auth middleware.

## Requirements

### Requirement: Register

The system MUST expose `POST /api/auth/register` accepting email, password, and username.

#### Scenario: Valid registration creates user and returns JWT

- GIVEN no user exists with email `a@b.com`
- WHEN `POST /api/auth/register` with `{email: "a@b.com", password: "Secret1!", username: "Alice"}`
- THEN response is HTTP 201 with `{token, user: {id, email, username}}`
- AND the password is stored as bcrypt hash
- AND a `users` row is created

#### Scenario: Duplicate email returns 409

- GIVEN user exists with email `a@b.com`
- WHEN the same email is used for registration
- THEN response is HTTP 409 with error message

### Requirement: Login

The system MUST expose `POST /api/auth/login` accepting email and password.

#### Scenario: Valid credentials return JWT

- GIVEN user exists with email `a@b.com` and password `Secret1!`
- WHEN `POST /api/auth/login` with correct credentials
- THEN response is HTTP 200 with `{token, user: {id, email, username}}`
- AND the JWT contains `sub` (user ID), `exp`, `iat`

#### Scenario: Invalid password returns 401

- GIVEN user exists with email `a@b.com`
- WHEN `POST /api/auth/login` with wrong password
- THEN response is HTTP 401

### Requirement: JWT Middleware

The system MUST require a valid JWT `Authorization: Bearer <token>` header on protected routes.

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
