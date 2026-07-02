# Design: Profile Picture Upload

## Technical Approach

Add two endpoints to `ProfileHandler` for multipart avatar upload/delete. Serve files via `e.Static()` outside DevMode guard. Client uploads with canvas resize (400×400) and `FormData` via a dedicated `uploadAvatar()` method that bypasses the JSON-only `request()`. Config drives paths and limits.

## Architecture Decisions

| Option | Tradeoffs | Decision |
|--------|-----------|----------|
| Config in ProfileHandler | Follows AuthHandler pattern (DB + Config already wired). Minimal wiring change. | Add `Config *config.Config` to ProfileHandler |
| e.Static() placement | If inside `if !cfg.DevMode {}`, uploads fail in prod. | Place before API groups, outside DevMode guard |
| Client upload method | `request()` hardcodes `application/json` — FormData needs no Content-Type. | New standalone `uploadAvatar(FormData)` using bare `fetch()` with Authorization header only |
| Extension source | Original filename can mismatch content. | Map detected MIME to extension: `image/jpeg→.jpg`, `image/png→.png`, `image/webp→.webp` |
| DB update failure → orphan | Writing file before DB risks orphan on DB error. | Write file → DB update → on DB failure `os.Remove(newFile)` |
| Startup dir creation | Manual mkdir is fragile. | `os.MkdirAll(cfg.UploadDir, 0755)` in handler init |
| `omitempty` on AvatarURL | nil field is absent from JSON, but DELETE contract requires `"avatar_url": null`. | **Remove `omitempty`** from struct tag. Backward-compatible: profiles without avatars now show `null` instead of absent. |
| Old file disk deletion | `avatar_url` is `/uploads/uuid.ext`. Need to map to disk path. | `filepath.Join(cfg.UploadDir, strings.TrimPrefix(oldURL, "/uploads/"))` |

## Data Flow

```
Browser                          Server                         Disk / DB
  │                                │                               │
  │── POST /api/profiles/me/avatar ──→ avatarUpload()              │
  │   multipart/form-data           │── os.MkdirAll if not exist   │
  │   field: "avatar"               │── http.MaxBytesReader        │
  │                                 │── r.FormFile("avatar")       │
  │                                 │── http.DetectContentType     │
  │                                 │── MIME→ext map (.jpg/.png/.webp)
  │                                 │── uuid + ext → newPath       │
  │                                 │── os.WriteFile(newPath)      │
  │                                 │── db UPDATE avatar_url=…     │
  │                                 │── on DB err: os.Remove(new)  │
  │                                 │── if old avatar is local:    │
  │                                 │   TrimPrefix + filepath.Join │
  │                                 │   → os.Remove(oldDiskPath)   │
  │←─ 200 {avatar_url, username…}   │                               │
```

## Data Model

No schema changes. `avatar_url TEXT` column already exists in `profiles` table.

## API Contracts

### POST /api/profiles/me/avatar

**Request**: `multipart/form-data`, field **`avatar`** — file (jpeg/png/webp, max 5MB)

**Response 200**:
```json
{"id":"uuid","username":"alice","email":"a@b.com","avatar_url":"/uploads/f4798e21-….jpg","status":"online","last_seen":"2026-07-01T12:00:00Z"}
```

**Errors**: 400 (no file), 413 (oversize), 415 (bad MIME), 500 (I/O or DB failure with orphan cleanup)

### DELETE /api/profiles/me/avatar

**Response 200** — same shape as GET, but `avatar_url` is now `null`:
```json
{"id":"uuid","username":"alice","email":"a@b.com","avatar_url":null,"status":"online","last_seen":"2026-07-01T12:00:00Z"}
```

### Fixed: `profileResponse` struct tag

```go
// BEFORE: AvatarURL *string `json:"avatar_url,omitempty"`  → nil = field absent
// AFTER:  AvatarURL *string `json:"avatar_url"`             → nil = "avatar_url": null
```

This changes existing GET/PATCH responses slightly: profiles without avatars will now emit `"avatar_url": null` instead of omitting the field. Fully backward-compatible — clients already handle nullable values.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `server/internal/config/config.go` | Modify | Add `UploadDir string`, `MaxFileSize int64`; env vars `UPLOAD_DIR`, `MAX_FILE_SIZE` |
| `server/internal/handlers/profile.go` | Modify | Add `Config *config.Config` to struct; **remove `omitempty`** from `AvatarURL`; add `avatarUpload()`, `avatarDelete()` methods; add MIME→ext map; add orphan cleanup; add dir init |
| `server/cmd/main.go` | Modify | Pass `cfg` to `ProfileHandler`; add `e.Static("/uploads", cfg.UploadDir)` outside DevMode; wire POST and DELETE routes |
| `client/src/lib/api.js` | Modify | Add `uploadAvatar(FormData)` using bare `fetch()` with only Authorization header; add `deleteAvatar()` reusing existing `request()` |
| `client/src/components/MainLayout.jsx` | Modify | Clickable current-user avatar → file picker → canvas resize → upload; show `<img>` when avatar_url set, initials fallback; delete overlay with confirmation |
| `client/src/components/ChatWindow.jsx` | Modify | Show contact avatar from `avatar_url`, fallback to first-letter initial when null |
| `infra/docker-compose.yml` | Modify | Add `./uploads:/app/uploads` volume mount to api service |

## Sequence Flow (Upload) — with ALL fixes applied

1. User clicks avatar → `hiddenInput.click()` → file picker (`accept: image/*`)
2. On select: validate size (<5MB) + type client-side
3. `new Image()` → draw to offscreen `<canvas>` at max 400×400 (maintain aspect ratio)
4. `canvas.toBlob()` → `FormData.append("avatar", blob, "avatar.jpg")`
5. Client calls `api.uploadAvatar(formData)` — dedicated method using bare `fetch()` with only `Authorization: Bearer <token>` header (no Content-Type override — browser sets `multipart/form-data; boundary=…` automatically)
6. Server: `os.MkdirAll(cfg.UploadDir, 0755)` at handler init (or on first call)
7. `http.MaxBytesReader(c.Request().Body, cfg.MaxFileSize + 512)` limits read
8. `r.FormFile("avatar")` → read first 512 bytes → `http.DetectContentType(buf)` → accept only `image/jpeg`/`image/png`/`image/webp`
9. Map detected MIME to extension: `image/jpeg→.jpg`, `image/png→.png`, `image/webp→.webp`
10. `uuid.NewString()` + extension → `newFile = filepath.Join(cfg.UploadDir, uuid+ext)`
11. Write file via `os.WriteFile(newFile, data, 0644)`
12. If previous `avatar_url` is non-null AND has prefix `/uploads/`: `oldDiskPath = filepath.Join(cfg.UploadDir, strings.TrimPrefix(oldURL, "/uploads/"))` → `os.Remove(oldDiskPath)`
13. `UPDATE profiles SET avatar_url = ? WHERE id = ?`
14. **If DB error**: `os.Remove(newFile)` → return 500. **Never leave orphan.**
15. Return full `profileResponse` with new `avatar_url`

## Sequence Flow (Delete)

1. Client calls `api.deleteAvatar()` — uses existing `request()` with method DELETE
2. Server: fetch current `avatar_url` from DB
3. If non-null and `/uploads/` prefix: map to disk path, `os.Remove()`
4. `UPDATE profiles SET avatar_url = NULL WHERE id = ?`
5. Return full `profileResponse` with `"avatar_url": null`

## Error Handling

| Scenario | HTTP | Code | Detail | Orphan cleanup |
|----------|------|------|--------|----------------|
| No file field | 400 | `MISSING_FILE` | "No avatar file provided" | N/A |
| Exceeds MaxFileSize | 413 | `FILE_TOO_LARGE` | "File exceeds maximum size" | N/A |
| Invalid MIME | 415 | `INVALID_MIME` | "Only JPEG, PNG, WebP accepted" | N/A |
| Disk write failure | 500 | `UPLOAD_FAILED` | "Failed to save file" | N/A |
| DB update fails | 500 | `INTERNAL` | "Failed to update profile" | `os.Remove(newFile)` |
| Delete, no avatar | 200 | — | No-op | N/A |
| Delete, file missing | 200 | — | No-op (DB still set to null) | N/A |

## Security Considerations

- **MIME spoofing**: `http.DetectContentType` uses magic bytes. Extension is derived from MIME, not user filename.
- **Path traversal**: UUID filenames + MIME-derived extension eliminate traversal.
- **File size**: `http.MaxBytesReader` enforced before body parsing.
- **Authorization**: Both endpoints behind `authmiddleware.AuthMiddleware`. User modifies own row only.
- **Write permissions**: `os.MkdirAll(0755)` at startup ensures directory exists and is writable.

## Dev vs Prod Differences

| Aspect | Dev | Prod |
|--------|-----|------|
| UploadDir | `./uploads` (project root) | `/app/uploads` (Docker) |
| Dir creation | `os.MkdirAll` in handler (dir may not exist) | Volume mount ensures /app/uploads exists |
| Serving | `e.Static()` outside DevMode | Same (not inside `if !cfg.DevMode {}`) |
| Env override | `UPLOAD_DIR=./dev-uploads` | `UPLOAD_DIR=/data/uploads` |

## Deployment (Dokploy)

**Prerequisite — Manual volume configuration in Dokploy:**
1. In the Dokploy dashboard for the DaveChat API service, go to **Storage** → **Add Volume**
2. Set: `Host path` = `/path/on/host/uploads` (or use a named volume), `Container path` = `/app/uploads`
3. The application expects `UPLOAD_DIR` env to match the container path (default `/app/uploads`)
4. If deploying without a volume: file uploads disappear on container restart — this is intentional (ephemeral dev) but not suitable for prod

## Migration / Rollout

No DB migration required. `avatar_url` column exists. New endpoints are additive. The `omitempty` removal is a serialization-only change (backward-compatible). Rollback: `git revert <commit>`.

## Open Questions

- None.
