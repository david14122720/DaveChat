# Database Specification

## Purpose

MariaDB schema and Go SQL driver replacing Supabase/PostgreSQL. Self-contained auth — no external `auth.users` dependency.

## Schema

| Table | Columns | PK | FK |
|-------|---------|----|----|
| `users` | `id CHAR(36)`, `email VARCHAR(255) UNIQUE`, `username VARCHAR(50) UNIQUE`, `password_hash VARCHAR(255)`, `avatar_url TEXT`, `status ENUM('online','offline','away')`, `last_seen DATETIME(3)`, `created_at DATETIME(3)` | `id` | — |
| `conversations` | `id CHAR(36)`, `created_at DATETIME(3)`, `last_message_at DATETIME(3)` | `id` | — |
| `participants` | `conversation_id CHAR(36)`, `user_id CHAR(36)`, `joined_at DATETIME(3)` | `(conversation_id, user_id)` | → `conversations(id)` ON DELETE CASCADE, → `users(id)` ON DELETE CASCADE |
| `messages` | `id CHAR(36)`, `conversation_id CHAR(36)`, `sender_id CHAR(36)`, `content TEXT`, `type ENUM('text','call')`, `created_at DATETIME(3)` | `id` | → `conversations(id)`, → `users(id)` ON DELETE CASCADE |

## Requirements

### Requirement: Schema Initialization

The system MUST apply the MariaDB schema on first startup via an init SQL script mounted in docker-compose.

#### Scenario: Fresh database initializes all tables

- GIVEN a new MariaDB container with no existing schema
- WHEN the init script runs
- THEN tables `users`, `conversations`, `participants`, `messages` are created
- AND `username_length` CHECK constraint requires username >= 3 chars

#### Scenario: Duplicate email or username is rejected

- GIVEN an existing user with email `a@b.com`
- WHEN a second user registers with the same email
- THEN MariaDB returns a duplicate entry error
- AND the Go handler returns HTTP 409

### Requirement: Go SQL Driver

The system MUST use `github.com/go-sql-driver/mysql` with a connection pool, replacing `pgx`.

#### Scenario: Connection pool connects to MariaDB

- GIVEN MariaDB is running on `DB_HOST:DB_PORT`
- WHEN `ConnectDB()` is called
- THEN a pool with 5-20 connections is established
- AND a ping query succeeds

### Requirement: docker-compose.yml

The system MUST define MariaDB + Go API server services in `docker-compose.yml`.

#### Scenario: `docker compose up` starts both services

- GIVEN no containers running
- WHEN `docker compose up` is executed
- THEN MariaDB starts on port 3306
- AND the Go server starts on port 8080
- AND the Go server connects to MariaDB successfully
