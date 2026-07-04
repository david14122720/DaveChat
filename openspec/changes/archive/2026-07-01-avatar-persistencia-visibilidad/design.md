# Design: Avatar Persistence & Visibility

## Technical Approach

Three independent concerns: (1) a Docker named volume in Dokploy to persist `/app/uploads` across redeploys; (2) conditional `<img>` rendering in the contact list sidebar replicating the ChatWindow pattern; (3) documentation of the volume setup in `dokploy.md`. The server already serves `/uploads` statically (line 76 of `main.go`) and the env var `UPLOAD_DIR=/app/uploads` is already set — the only gap is the volume mount itself.

## Architecture Decisions

### Decision: Dokploy Volume Type

| Option | Tradeoff |
|--------|----------|
| **Named volume** (chosen) | Docker-managed lifecycle; consistent with existing MariaDB volume `davechat-mariadb-66rn9m-data`; survives container recreation; no host path dependency. |
| Host bind mount | Requires manual host directory creation; fragile across Dokploy server migrations. |

**Volume name**: `davechat-m3bjt5-uploads` → `/app/uploads`

### Decision: Contact List Avatar Dimensions

| Option | Tradeoff |
|--------|----------|
| **Change to w-10 h-10 rounded-full** (chosen) | Matches ChatWindow header exactly; consistent avatar sizing across all surfaces. |
| Keep w-12 h-12 rounded-2xl | Inconsistent with ChatWindow; different size per surface creates visual noise. |

The contact list avatar changes from a `w-12 h-12 rounded-2xl` initials-only circle to `w-10 h-10 rounded-full` with conditional `<img>` — same as ChatWindow lines 137-148.

### Decision: Loading State

No skeleton or fade-in. Contact avatars are small (40×40px) and will load near-instantly from the same origin. The img `onError` handler is not needed because the server already returns 404 for missing files and the parent checks `avatar_url` truthiness.

## Data Flow

```
User uploads avatar → POST /api/profiles/me/avatar → saves file to /app/uploads/{filename}
                                                           │
                                                    (Dokploy named volume)
                                                           │
                                                      survives redeploy
                                                           │
Profile response → { avatar_url: "/uploads/filename.jpg" }
                        │
             ┌──────────┼──────────┐
             │          │          │
        MainLayout  ChatWindow  Profile Modal
        (sidebar)   (header)    (user avatar)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `client/src/components/MainLayout.jsx` | Modify | Contact list avatar: conditional `<img>` when `avatar_url` is set, matching ChatWindow pattern |
| `dokploy.md` | Modify | Add "Volume Setup" section documenting the named volume creation and sequence |
| Dokploy app `_-BbzRdJ18Cuh6EXslZvQ` | Modify | Add named volume `davechat-m3bjt5-uploads` at `/app/uploads` (via Dokploy UI, not code) |

### MainLayout.jsx — Contact Avatar Change

**Before** (lines 253-258):
```jsx
<div className="relative shrink-0">
  <div className="w-12 h-12 rounded-2xl bg-slate-800 border border-slate-700 flex items-center justify-center font-bold text-slate-300 shadow-sm">
    {contact.username[0].toUpperCase()}
  </div>
  <div className={'absolute -bottom-1 -right-1 w-3.5 h-3.5 border-2 border-slate-900 rounded-full shadow-lg ' + (isOnline(contact) ? 'bg-emerald-400' : 'bg-slate-600')}></div>
</div>
```

**After**:
```jsx
<div className="relative shrink-0">
  {contact.avatar_url ? (
    <img
      src={contact.avatar_url.startsWith('http') ? contact.avatar_url : `${import.meta.env.VITE_API_URL || ''}${contact.avatar_url}`}
      alt={contact.username}
      className="w-10 h-10 rounded-full object-cover border border-slate-600 shadow-inner"
    />
  ) : (
    <div className="w-10 h-10 rounded-full bg-gradient-to-br from-slate-700 to-slate-800 border border-slate-600 flex items-center justify-center font-bold text-slate-200 shadow-inner">
      {contact.username[0].toUpperCase()}
    </div>
  )}
  <div className={'absolute -bottom-1 -right-1 w-3.5 h-3.5 border-2 border-slate-900 rounded-full shadow-lg ' + (isOnline(contact) ? 'bg-emerald-400' : 'bg-slate-600')}></div>
</div>
```

The change reduces avatar size from 48px to 40px (matching ChatWindow) and changes shape from `rounded-2xl` (rounded square) to `rounded-full` (circle). The online indicator position stays identical relative to the container.

## Dokploy Volume Setup (UI Steps)

1. Navigate to **Aplicaciones Web → production → DaveChat** in Dokploy
2. Go to **Volumes** tab → **Add Volume**
3. Set **Volume name**: `davechat-m3bjt5-uploads`
4. Set **Mount path**: `/app/uploads`
5. **Save** — this creates the Docker named volume
6. **Do NOT redeploy yet** — the volume must exist first
7. (Optional) Upload a test file to verify persistence before triggering a redeploy

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | MainLayout contact rendering | Assert `<img>` renders when `avatar_url` is set; initials fallback when null |
| Manual | Volume persistence | Upload an avatar, trigger Dokploy redeploy, verify avatar still loads at `/uploads/*` |
| Manual | Contact list layout | Verify contacts with and without avatars render correctly on mobile and desktop |

## Migration / Rollout

No data migration script. Before first redeploy with the volume attached, existing uploads in the container's writable layer will be lost. Mitigation: configure the volume first, then redeploy. Any uploads created after volume setup survive permanently.

## Risk Mitigation — Configure-Before-Redeploy

The critical sequence is: **(1) add volume → (2) verify → (3) redeploy**. Reversing this loses existing uploads. Dokploy stores volume configuration in its own database, so adding the volume via the UI is a metadata-only operation — no container restart happens until the next deploy. This makes the sequence safe by design: the volume is registered with Dokploy, then activated on the next deployment.

## Open Questions

- [ ] Should we add an `onError` fallback to initials if the image URL returns a broken response? (low priority, 404s already show browser broken-img; can add later if users report it)
