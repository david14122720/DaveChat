# Proposal: Echo Migration + DB Enhancements

## Intent

Unificar el stack del servidor Go en un solo binario (Echo + embed) y agregar funcionalidades faltantes: refresh tokens, read receipts, call logs y búsqueda fulltext. El driver principal es simplificar el deploy — un solo binario sin depender de servidor de desarrollo para el frontend.

## Scope

### In Scope
- DB: tablas `refresh_tokens`, `read_receipts`, `call_logs` + FULLTEXT index en `messages.content`
- Router: migrar chi v5 → Echo v4 manteniendo rutas y middleware
- Refresh token: rotation style, endpoint `POST /api/auth/refresh`
- Read receipts: granularidad por mensaje, endpoint `POST /api/conversations/{id}/read`
- Call logs: registro de llamadas con tipo/status/duración
- Search: `GET /api/conversations/{id}/messages/search?q=termino`
- Single binary: `//go:embed` del build de Vite en el binario Go

### Out of Scope
- WebSocket sobre Echo (mantener gorilla/websocket independiente)
- Edición/borrado de mensajes
- Traer conversaciones archivadas
- Reacciones a mensajes
- UI de read receipts, call logs o search (solo API)

## Capabilities

### New Capabilities
- `call-logs`: registro y consulta de llamadas (audio/video) en la tabla `call_logs`
- `message-search`: endpoint FULLTEXT con BOOLEAN MODE sobre `messages.content`
- `single-binary-embed`: servidor de archivos estáticos embebidos desde Vite build

### Modified Capabilities
- `database`: nuevas tablas `refresh_tokens`, `read_receipts`, `call_logs` + FULLTEXT index en `messages`
- `user-auth`: refresh token rotation, nuevo endpoint `/api/auth/refresh`
- `api-gateway`: router chi→Echo v4, nuevas rutas de search/read/call-logs/static, embed del build frontend
- `messages`: nuevo endpoint search, nueva tabla `read_receipts` para tracking de lectura

## Approach

3 fases secuenciales, cada una commit independiente:

**Fase 1 — DB schema**: Agregar tablas `refresh_tokens`, `read_receipts`, `call_logs` al `01-schema.sql`. Agregar FULLTEXT index en `messages.content`. Sin cambios en handlers — solo schema.

**Fase 2 — Chi → Echo**: Reemplazar chi.NewRouter() por echo.New(). Migrar handlers a `echo.Context`. Migrar middleware (AuthMiddleware → echo middleware). Mantener gorilla/websocket en ruta aparte (Echo usa `http.Handler` wrapper). Refactor por archivo, probar ruta por ruta.

**Fase 3 — Single binary**: Actualizar Vite config para build relativo. Usar `//go:embed client/dist/*` en Go. Agregar `StaticFileServer` middleware en Echo para fallback SPA. Remover variables de `ALLOWED_ORIGINS` y `VITE_API_URL`.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `db/init/01-schema.sql` | Modified | New tables + FULLTEXT index |
| `server/cmd/main.go` | Rewrite | Echo router, embed FS, refresh routes |
| `server/internal/handlers/*.go` | Modified | Convert chi to echo.Context, add refresh/search/read handlers |
| `server/internal/middleware/auth.go` | Modified | Adapt to echo middleware signature |
| `server/internal/config/config.go` | Modified | Remove ALLOWED_ORIGINS, add StaticDir |
| `client/vite.config.js` | Modified | Build output to relative `dist/` |
| `server/Dockerfile` | Modified | Add `COPY --from=build client/dist` |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Echo middleware behavior differs from chi on edge cases | Low | Test each route manually after migration; preserve chi branch as rollback |
| Embed increases binary size | Low | ~2MB for Vite build; acceptable. Use `-s -w` ldflags |
| Refresh token rotation breaks existing sessions | Medium | Ship Phase 1 DB schema first, then deploy refresh token logic; old JWTs still valid during transition |

## Rollback Plan

Por fase:
- **Fase 1**: `git revert <phase1-commit>` — additive schema, safe to reverse
- **Fase 2**: `git revert <phase2-commit>` — mantiene el branch chi como `refs/changes/chi-router` para restore rápido
- **Fase 3**: `git revert <phase3-commit>` + redeploy sin embed

## Dependencies

- Echo v4 (`github.com/labstack/echo/v4`)
- Vite build debe existir antes de compilar el binario (Fase 3)

## Success Criteria

- [ ] Server compila y corre con Echo v4, mismas rutas respondiendo igual que con chi
- [ ] `POST /api/auth/refresh` emite nuevo token e invalida el anterior
- [ ] `POST /api/conversations/{id}/read` inserta read_receipt y no duplica
- [ ] `GET /api/conversations/{id}/messages/search?q=termino` devuelve resultados FULLTEXT
- [ ] Llamadas se persisten en `call_logs` con tipo/status/duración
- [ ] `GET /` sirve el frontend embebido sin servidor externo
- [ ] Fases 1-3: cada fase compila y pasa `go build ./...`
