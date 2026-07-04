# Tasks: Avatar Persistence & Visibility

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~35–50 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | auto-forecast |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

## Prerequisites

- [ ] Dokploy admin access to application `_-BbzRdJ18Cuh6EXslZvQ`
- [ ] Verify `UPLOAD_DIR=/app/uploads` is set in Dokploy env vars

## Implementation Tasks

### Phase 1: Documentation
- [x] 1.1 Add "Volume Setup" section to `dokploy.md` after the MariaDB section: volume name `davechat-m3bjt5-uploads`, mount path `/app/uploads`, configure-before-redeploy sequence, and Dokploy UI steps from design.md lines 90–96

### Phase 2: Frontend — Contact List Avatar
- [x] 2.1 In `MainLayout.jsx` lines 253–258: replace initials-only `<div>` with conditional render — `<img>` when `contact.avatar_url` is truthy, initials fallback when null
- [x] 2.2 Match ChatWindow.jsx line 139 URL construction: `contact.avatar_url.startsWith('http') ? contact.avatar_url : \`${import.meta.env.VITE_API_URL || ''}${contact.avatar_url}\``
- [x] 2.3 Match ChatWindow.jsx line 141 dimensions: `w-10 h-10 rounded-full object-cover border border-slate-600 shadow-inner`
- [x] 2.4 Keep online indicator div unchanged (design.md line 82)

### Phase 3: Infrastructure (Dokploy UI)
- [ ] 3.1 Dokploy → Aplicaciones Web → production → DaveChat → Volumes → Add Volume: name `davechat-m3bjt5-uploads`, mount path `/app/uploads`, save
- [ ] 3.2 Verify volume appears in Volumes list before redeploying
- [ ] 3.3 Redeploy the application to activate the volume mount

### Phase 4: Verification
- [ ] 4.1 Upload a test avatar, trigger redeploy, verify the avatar still loads at `/uploads/*`
- [ ] 4.2 Verify contacts with and without avatars render correctly in the sidebar
