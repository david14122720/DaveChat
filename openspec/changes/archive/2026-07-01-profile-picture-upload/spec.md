# Spec: Profile Picture Upload

## ADDED Requirements — Avatar Upload

### Requirement: Upload Avatar (POST)

The system MUST expose `POST /api/profiles/me/avatar` accepting `multipart/form-data` with an `avatar` field. MUST validate MIME type (JPEG, PNG, WebP) via `http.DetectContentType`, MUST reject files over `MAX_FILE_SIZE`, MUST save with a UUID filename, MUST update the requesting user's `avatar_url`, and MUST delete the previous file from disk if replacing.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Valid upload succeeds | Alice with JWT, no existing avatar | POST with valid JPEG <5MB | 200 + `{avatar_url: "/uploads/<uuid>.jpg"}`, file on disk |
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

The client MUST provide a clickable avatar that opens a file picker (`accept: image/*`). On selection, the image MUST be loaded onto a canvas, resized to 400×400 (maintaining aspect ratio), converted to Blob (JPEG), and submitted via `FormData` to the upload endpoint. The avatar display MUST render an `<img>` when `avatar_url` is set and an initials fallback when null.

### Requirement: Configuration

The system MUST read `UPLOAD_DIR` env var (default `/app/uploads`) for the disk path and `MAX_FILE_SIZE` env var (default `5242880` — 5MB) for the size limit.

## MODIFIED Requirements — User Profiles

### Requirement: Get Own Profile

Same behavior — `GET /api/profiles/me` returns `avatar_url`. The field SHALL now reflect the uploaded file path when set via the upload endpoint.
(Previously: avatar_url only settable via PATCH JSON body)

#### Scenario: Returns current user (unchanged)

- GIVEN a valid JWT for user Alice
- WHEN `GET /api/profiles/me` is called
- THEN response is HTTP 200 with `{id, email, username, avatar_url, status, last_seen}`
- AND `avatar_url` reflects the path from the most recent upload

### Requirement: Update Profile

Same behavior — `PATCH /api/profiles/me` still accepts `avatar_url` via JSON body. The upload endpoint is an ADDITIONAL mechanism.
(Previously: PATCH was the only way to set avatar_url)

#### Scenario: Valid update succeeds (unchanged)

- GIVEN a valid JWT for user Alice
- WHEN `PATCH /api/profiles/me` with `{username: "AliceNew"}`
- THEN response is HTTP 200 with updated user

#### Scenario: Duplicate username returns 409 (unchanged)

- GIVEN Bob already has username "Bob"
- WHEN Alice tries to update her username to "Bob"
- THEN response is HTTP 409