# Proposal: Profile Picture Upload

## Intent

Users need a profile photo. The DB already has `avatar_url` in profiles but only accepts raw URLs via PATCH — no upload flow. This adds a proper multipart upload workflow with client-side resize, server-side validation, and static serving.

## Scope

### In Scope
- Backend: `POST /api/profiles/me/avatar` — multipart, MIME validation (jpeg/png/webp), size limit (5MB), UUID filename, writes to disk, sets `avatar_url`
- Backend: `DELETE /api/profiles/me/avatar` — nulls `avatar_url`, deletes file from disk
- Backend: static serving via `e.Static("/uploads", uploadDir)` in main.go
- Config: `UploadDir` (default `/app/uploads`) and `MaxFileSize` (default 5MB)
- Client: clickable avatar → hidden file input → canvas resize (400×400) → FormData upload
- Client: render `<img>` when `avatar_url` set, initials fallback when null
- Dev: docker-compose volume mount for uploads

### Out of Scope
- Cropping or editor UI
- CDN or cloud storage (S3, Cloudinary, etc.)
- Animated avatars (GIF/AVIF)
- Upload history or gallery
- Default avatar generation beyond initials

## Capabilities

### New Capabilities
- `avatar-upload`: file reception, MIME/size validation, UUID storage on disk, static serving, and cleanup on delete

### Modified Capabilities
- `user-profiles`: add upload/delete endpoints; `avatar_url` now settable via file upload in addition to PATCH

## Approach

1. Add `UploadDir` and `MaxFileSize` to `Config` struct
2. New handler `avatarUpload`: parse multipart → validate MIME via `http.DetectContentType` → check size → generate UUID filename → write to disk → update DB row → delete old file if exists
3. New handler `avatarDelete`: null `avatar_url` in DB → remove file from disk
4. Wire routes: `POST /api/profiles/me/avatar` and `DELETE /api/profiles/me/avatar`
5. Add `e.Static("/uploads", uploadDir)` in `main.go`
6. Client: avatar element → `<input type="file" accept="image/*">` → load into canvas → `canvas.toBlob()` → `FormData` → fetch POST
7. docker-compose: add uploads volume mount

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `server/internal/config/config.go` | Modified | Add UploadDir, MaxFileSize |
| `server/internal/handlers/profile.go` | Modified | Add upload/delete handlers |
| `server/internal/handlers/routes.go` | Modified | Wire new endpoints |
| `server/cmd/main.go` | Modified | Add e.Static for uploads |
| `client/src/components/MainLayout.jsx` | Modified | Avatar with file picker + canvas resize |
| `docker-compose.yml` | Modified | Volume mount for uploads |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| MIME spoofing via magic bytes | Low | Server validates `http.DetectContentType`, not extension |
| Path traversal in filename | Low | UUID filenames eliminate traversal risk |
| Volume not configured in Dokploy | Med | Document manual step in deploy instructions |
| Old avatar orphaned on upload | Low | Delete old file synchronously before new write |

## Rollback Plan

`git revert <commit>` — fully additive endpoints, no DB migration needed. Static serving toggle is in config. Previous PATCH behavior unchanged.

## Dependencies

- Echo v4 `e.Static()` for file serving
- Docker volume mount in Dokploy (manual configuration step)

## Success Criteria

- [ ] `POST /api/profiles/me/avatar` with valid image returns 200 and sets `avatar_url`
- [ ] `POST /api/profiles/me/avatar` with >5MB file returns 413
- [ ] `POST /api/profiles/me/avatar` with invalid MIME returns 415
- [ ] `DELETE /api/profiles/me/avatar` returns 200 with `avatar_url: null`
- [ ] Uploaded file is accessible at `/uploads/<uuid>.<ext>`
- [ ] Client shows uploaded image; clicking avatar opens file picker
- [ ] `go build ./...` passes
