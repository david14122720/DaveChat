# Proposal: Avatar Persistence & Visibility

## Intent

Two problems: (1) uploaded avatars vanish on every Dokploy redeploy because no persistent volume is mounted at `/app/uploads`; (2) the sidebar contact list shows initials instead of avatars, inconsistent with the ChatWindow header and the WhatsApp-like experience users expect.

## Scope

### In Scope
- Configure persistent Docker volume on Dokploy for `/app/uploads`
- Update sidebar contact list to render `<img>` tags when `avatar_url` is set
- Update `dokploy.md` with volume setup and maintenance docs

### Out of Scope
- Avatar upload flow (already works)
- Group chat avatars
- Image compression or optimization
- Migration of existing uploaded files from current container
- Avatar upload UI improvements

## Capabilities

### New Capabilities
None

### Modified Capabilities
- `user-profiles`: Extend client-side avatar display requirement to cover the contact list (sidebar) — render `<img>` when `avatar_url` is set, initials fallback when null

## Approach

1. **Dokploy volume**: Add a named volume in the Dokploy application config, mounting a persistent host path to `/app/uploads`. Survives redeploys and container recreation.
2. **Contact list avatars**: In `MainLayout.jsx`, iterate the contact list and render `<img src={profile.avatar_url} />` when truthy, falling back to the existing initials-in-circle. Follow the same pattern already in `ChatWindow.jsx`.
3. **Docs**: Update `dokploy.md` with volume creation steps, Dokploy UI configuration, and data persistence notes.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| Dokploy app config (`_-BbzRdJ18Cuh6EXslZvQ`) | Modified | Add persistent volume mount for `/app/uploads` |
| `client/src/components/MainLayout.jsx` | Modified | Render avatars in contact list |
| `dokploy.md` | Modified | Volume setup and maintenance docs |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Existing avatars lost on first redeploy before volume is configured | High | Document sequence: configure volume first, then redeploy |
| Wrong volume path breaks file serving | Low | Verify with test upload after volume setup |
| Breaking contact list layout with avatar images | Low | Use same dimensions as initials circle, test responsiveness |

## Rollback Plan

Remove the volume mount in Dokploy settings and redeploy. For the contact list, revert JSX changes in `MainLayout.jsx`.

## Dependencies

- Dokploy admin access (confirmed)
- Application ID: `_-BbzRdJ18Cuh6EXslZvQ`

## Success Criteria

- [ ] New avatar uploads survive a Dokploy redeploy
- [ ] Contact list shows avatar images when available, initials fallback when not
- [ ] No regressions in ChatWindow avatar display
