# Tasks: Profile Picture Upload

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~190 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

## Phase 1: Foundation — Config + Docker

- [x] 1.1 Add `UploadDir string` and `MaxFileSize int64` fields to `Config` struct in `server/internal/config/config.go` with `UPLOAD_DIR` (default `/app/uploads`) and `MAX_FILE_SIZE` (default `5242880`) env vars
- [x] 1.2 Add `./uploads:/app/uploads` volume mount to api service in `infra/docker-compose.yml`

## Phase 2: Core Server — Upload & Delete Handlers

- [x] 2.1 Add `Config *config.Config` to `ProfileHandler` struct; remove `omitempty` from `AvatarURL` json tag in `profileResponse`; add MIME-to-extension map constant and `os.MkdirAll(cfg.UploadDir, 0755)` init in `server/internal/handlers/profile.go`
- [x] 2.2 Implement `avatarUpload(c echo.Context) error`: `http.MaxBytesReader` → `r.FormFile("avatar")` → `http.DetectContentType` validation → UUID+MIME-ext filename → `os.WriteFile` → DB UPDATE → orphan cleanup on DB failure → old file deletion → return `profileResponse`
- [x] 2.3 Implement `avatarDelete(c echo.Context) error`: fetch current `avatar_url` from DB → delete file from disk (if local path) → SET `avatar_url = NULL` → return `profileResponse`

## Phase 3: Integration — Routes + Client API

- [x] 3.1 Wire `POST /api/profiles/me/avatar` and `DELETE /api/profiles/me/avatar` routes; add `e.Static("/uploads", cfg.UploadDir)` outside DevMode guard; pass `cfg` to `ProfileHandler` in `server/cmd/main.go`
- [x] 3.2 Add `uploadAvatar(formData)` using bare `fetch()` with auth header only (no Content-Type) and `deleteAvatar()` using existing `request()` with DELETE method in `client/src/lib/api.js`

## Phase 4: Client UI — Avatar Components

- [x] 4.1 Add clickable user avatar → hidden file input with client-side type+size validation → canvas resize to 400×400 → FormData upload via `api.uploadAvatar()`; show `<img>` when `avatar_url` set, initials fallback; add delete overlay (trash icon on hover + confirmation) with local user state update in `client/src/components/MainLayout.jsx`
- [x] 4.2 Show contact avatar from `contact.avatar_url` with initial-letter fallback when null in `client/src/components/ChatWindow.jsx`

## Phase 5: Verification

- [x] 5.1 Run `go build ./...` and `cd client && npm run build` to verify compilation; smoke-test upload/delete flows against running server
