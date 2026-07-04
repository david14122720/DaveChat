# User Profiles Specification

## Purpose

REST handlers for reading and updating user profiles. Profiles are stored in the `users` table (merged with auth — no separate profiles table). Supports avatar upload and deletion via dedicated endpoints in addition to JSON PATCH.

## Requirements

### Requirement: Get Own Profile

The system MUST expose `GET /api/profiles/me` returning the authenticated user's profile. The `avatar_url` field SHALL reflect the path from the most recent upload (or PATCH) when set; null when none.

#### Scenario: Returns current user

- GIVEN a valid JWT for user Alice
- WHEN `GET /api/profiles/me` is called
- THEN response is HTTP 200 with `{id, email, username, avatar_url, status, last_seen}`
- AND `avatar_url` reflects the path from the most recent upload or PATCH

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

The system MUST expose `PATCH /api/profiles/me` for updating username and avatar_url. The upload endpoint (`POST /api/profiles/me/avatar`) is an ADDITIONAL mechanism for setting `avatar_url` via file upload.

#### Scenario: Valid update succeeds

- GIVEN a valid JWT for user Alice
- WHEN `PATCH /api/profiles/me` with `{username: "AliceNew"}`
- THEN response is HTTP 200 with updated user
- AND the username is changed in the database

#### Scenario: Duplicate username returns 409

- GIVEN Bob already has username "Bob"
- WHEN Alice tries to update her username to "Bob"
- THEN response is HTTP 409

### Requirement: Upload Avatar (POST)

The system MUST expose `POST /api/profiles/me/avatar` accepting `multipart/form-data` with an `avatar` field. MUST validate MIME type (JPEG, PNG, WebP) via `http.DetectContentType`, MUST reject files over `MAX_FILE_SIZE`, MUST save with a UUID filename, MUST update the requesting user's `avatar_url`, and MUST delete the previous file from disk if replacing.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Valid upload succeeds | Alice with JWT, no existing avatar | POST with valid JPEG \<5MB | 200 + `{avatar_url: "/uploads/\<uuid\>.jpg"}`, file on disk |
| Replace existing | Alice has `avatar_url = "/uploads/old.jpg"` | POST with new valid PNG | Old file deleted, 200 + new URL |
| Wrong MIME returns 415 | Valid JWT | POST with GIF file | HTTP 415 |
| Too large returns 413 | Valid JWT | POST with 6MB file | HTTP 413 |
| No file returns 400 | Valid JWT | POST with no file field | HTTP 400 |
| MIME spoofing prevented | File with `.jpg` ext but PNG magic bytes | POST to upload | Detected as PNG via `http.DetectContentType`, accepted |

### Requirement: Delete Avatar (DELETE)

The system MUST expose `DELETE /api/profiles/me/avatar`. MUST set `avatar_url` to NULL in the DB and delete the file from disk.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Existing avatar deleted | Alice has `avatar_url = "/uploads/pic.jpg"` | DELETE /api/profiles/me/avatar | 200 + `{avatar_url: null}`, file deleted |
| No avatar to delete | Alice has `avatar_url = null` | DELETE /api/profiles/me/avatar | 200 + `{avatar_url: null}`, no-op |

### Requirement: Static File Serving

The system MUST serve uploaded files by registering `e.Static("/uploads", uploadDir)`.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| File is served | File `abc.jpg` exists in uploadDir | GET /uploads/abc.jpg | 200 with correct Content-Type |
| Missing file | File does not exist | GET /uploads/nonexistent.jpg | 404 |

### Requirement: Client-Side Image Upload

The client MUST provide a clickable avatar that opens a file picker (`accept: image/*`). On selection, the image MUST be loaded onto a canvas, resized to 400×400 (maintaining aspect ratio), converted to Blob (JPEG), and submitted via `FormData` to the upload endpoint. Avatar display in ALL surfaces — profile page, ChatWindow header, and contact list sidebar — MUST render an `<img>` when `avatar_url` is set and an initials-in-circle fallback when null. The contact list MUST use the same dimensions (`w-10 h-10`) and style (`rounded-full`, `object-cover`) as the existing ChatWindow pattern to avoid layout shifts.

#### Scenario: Contact with avatar

- GIVEN a contact has `avatar_url` set to a valid URL
- WHEN the sidebar contact list renders
- THEN the contact shows an `<img>` tag with that URL at `w-10 h-10 rounded-full object-cover`

#### Scenario: Contact without avatar

- GIVEN a contact has `avatar_url` set to null
- WHEN the sidebar contact list renders
- THEN the contact shows the initials-in-circle fallback (existing behavior, unchanged)

#### Scenario: ChatWindow unaffected

- GIVEN ChatWindow already renders avatars
- WHEN this change is deployed
- THEN ChatWindow avatar display is unchanged

### Requirement: Configuration

The system MUST read `UPLOAD_DIR` env var (default `/app/uploads`) for the disk path and `MAX_FILE_SIZE` env var (default `5242880` — 5MB) for the size limit.
