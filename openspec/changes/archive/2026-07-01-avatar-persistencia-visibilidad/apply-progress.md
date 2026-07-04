# Apply Progress: Avatar Persistence & Visibility

## Completed Tasks

### Phase 1: Documentation
- [x] 1.1 Add "Volume Setup" section to `dokploy.md` — volume name `davechat-m3bjt5-uploads`, mount path `/app/uploads`, configure-before-redeploy sequence, verify steps

### Phase 2: Frontend — Contact List Avatar
- [x] 2.1 Conditional render in MainLayout.jsx — `<img>` when `contact.avatar_url` is truthy, initials fallback when null
- [x] 2.2 URL construction matches ChatWindow: `contact.avatar_url.startsWith('http') ? contact.avatar_url : \`${import.meta.env.VITE_API_URL || ''}${contact.avatar_url}\``
- [x] 2.3 Dimensions match ChatWindow: `w-10 h-10 rounded-full object-cover border border-slate-600 shadow-inner`
- [x] 2.4 Online indicator div unchanged

### Phase 3: Infrastructure (Dokploy UI)
- [ ] 3.1 Add volume in Dokploy UI — requires manual action
- [ ] 3.2 Verify volume appears in Volumes list
- [ ] 3.3 Redeploy the application

### Phase 4: Verification
- [ ] 4.1 Upload test avatar, redeploy, verify persists
- [ ] 4.2 Verify contacts with/without avatars render correctly

## Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `dokploy.md` | Modified | Added "Volume Setup" section after MariaDB section with Dokploy UI steps and persistence verification |
| `client/src/components/MainLayout.jsx` | Modified | Contact list avatar: replaced fixed initials div with conditional `<img>`/initials fallback matching ChatWindow pattern |

## Deviations from Design

None — implementation matches design exactly.

## Build Verification

| Layer | Status |
|-------|--------|
| `go build ./...` | ✅ Passed |
| `npm run build` | ✅ Passed (1653 modules, 7.58s) |

## Remaining Manual Steps

1. Dokploy UI: Navigate to Aplicaciones Web → production → DaveChat → Volumes → Add Volume → name `davechat-m3bjt5-uploads`, mount path `/app/uploads`
2. Verify volume appears in list before redeploying
3. Redeploy to activate the mount
4. Upload a test avatar, trigger redeploy, verify it persists
5. Verify contacts with and without avatars render correctly in the sidebar
