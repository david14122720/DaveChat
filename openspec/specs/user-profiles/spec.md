# User Profiles Specification

## Purpose

REST handlers for reading and updating user profiles. Profiles are stored in the `users` table (merged with auth — no separate profiles table).

## Requirements

### Requirement: Get Own Profile

The system MUST expose `GET /api/profiles/me` returning the authenticated user's profile.

#### Scenario: Returns current user

- GIVEN a valid JWT for user Alice
- WHEN `GET /api/profiles/me` is called
- THEN response is HTTP 200 with `{id, email, username, avatar_url, status, last_seen}`

### Requirement: List All Profiles

The system MUST expose `GET /api/profiles` returning all users except the authenticated one.

#### Scenario: Returns excluding self

- GIVEN three users exist (Alice, Bob, Charlie)
- WHEN Alice calls `GET /api/profiles`
- THEN response is HTTP 200 with array containing Bob and Charlie
- AND Alice is excluded from results

#### Scenario: Empty list when alone

- GIVEN only one user exists (Alice)
- WHEN Alice calls `GET /api/profiles`
- THEN response is HTTP 200 with empty array

### Requirement: Update Profile

The system MUST expose `PATCH /api/profiles/me` for updating username and avatar_url.

#### Scenario: Valid update succeeds

- GIVEN a valid JWT for user Alice
- WHEN `PATCH /api/profiles/me` with `{username: "AliceNew"}`
- THEN response is HTTP 200 with updated user
- AND the username is changed in the database

#### Scenario: Duplicate username returns 409

- GIVEN Bob already has username "Bob"
- WHEN Alice tries to update her username to "Bob"
- THEN response is HTTP 409
