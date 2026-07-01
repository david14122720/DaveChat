# DaveChat

Sistema de mensajería en tiempo real con soporte de llamadas de audio/video vía WebRTC. Backend en Go + Echo, frontend en React + Vite + TailwindCSS, base de datos MariaDB.

## Stack

| Capa | Tecnología |
|------|-----------|
| **Backend** | Go 1.26, Echo v4, Gorilla WebSocket, JWT (golang-jwt) |
| **Frontend** | React 18, Vite 5, TailwindCSS 3, Framer Motion, Lucide Icons |
| **Base de datos** | MariaDB (MySQL) |
| **Tiempo real** | WebSocket propio + WebRTC (RTCPeerConnection) |
| **Autenticación** | JWT + Refresh Tokens con rotación |
| **Infraestructura** | Docker, Dokploy |

## Funcionalidades

- **Chat en tiempo real**: mensajes con polling 2s, orden WhatsApp-style (más viejos arriba, más nuevos abajo)
- **Llamadas de audio/video**: signaling vía WebSocket, peer-to-peer vía WebRTC con STUN
- **Presencia en línea**: detección de estado online/offline con timeout de 90s
- **Historial de llamadas**: registro de llamadas completadas, perdidas, rechazadas y canceladas
- **Autenticación**: registro, login, refresh token con rotación y hash SHA-256
- **Búsqueda de mensajes**: full-text search en contenido de mensajes

## Estructura del proyecto

```
├── client/                  # Frontend React + Vite
│   ├── src/
│   │   ├── components/      # AuthLayout, MainLayout, ChatWindow, CallUI, etc.
│   │   ├── contexts/        # CallContext (WebRTC state machine)
│   │   ├── lib/             # api.js (REST client), websocket.js (WS client)
│   │   └── hooks/           # useAuth (auth state)
│   └── package.json
├── server/                  # Backend Go
│   ├── cmd/
│   │   └── main.go          # Entry point con Echo + embedded frontend
│   └── internal/
│       ├── config/          # Config desde variables de entorno
│       ├── database/        # Conexión MariaDB
│       ├── handlers/        # Auth, Profile, Conversation, Message, CallLog
│       ├── middleware/       # JWT auth middleware
│       └── websocket/       # Hub + Handler (WebRTC signaling)
├── infra/
│   └── mariadb/init.sql     # Schema completo de base de datos
├── Dockerfile               # Multi-stage build (Node 20 → Go 1.25 → Alpine)
└── .env                     # Variables de entorno
```

## Desarrollo local

```bash
# 1. Frontend (terminal 1)
cd client
npm install
npm run dev

# 2. Backend (terminal 2)
export JWT_SECRET=tu-secreto-aqui
export DEV_MODE=true       # Habilita CORS para localhost:5173
cd server && go run ./cmd/main.go
```

> La base de datos MariaDB corre en Dokploy (`192.168.101.133:3307`). Ver `dokploy.md` para más detalles.

## Despliegue

El deploy se hace vía **Dokploy** (ver `dokploy.md`).

### Local con Docker Compose

```bash
docker compose -f infra/docker-compose.yml up -d --build
```

> La base de datos corre en Dokploy, no se necesita levantar MariaDB local.

## Variables de entorno

| Variable | Descripción | Default |
|----------|------------|---------|
| `JWT_SECRET` | Secreto para firmar JWT (requerido) | — |
| `PORT` | Puerto del servidor | `8080` |
| `DB_DSN` | DSN de conexión a MariaDB | `davechat:davechat_pass@tcp(192.168.101.133:3307)/davechat?parseTime=true&...` |
| `ALLOWED_ORIGINS` | Orígenes permitidos para CORS | `http://localhost:5173` |
| `DEV_MODE` | Modo desarrollo (true = CORS habilitado, sin embed) | `false` |

## API endpoints

### Públicos
- `POST /api/auth/register` — Registro de usuario
- `POST /api/auth/login` — Inicio de sesión
- `POST /api/auth/refresh` — Refresh de token

### Protegidos (JWT Bearer)
- `GET /api/auth/me` — Perfil del usuario autenticado
- `GET /api/profiles` — Lista de contactos
- `GET /api/profiles/:id` — Perfil por ID
- `GET /api/conversations` — Conversaciones del usuario
- `POST /api/conversations` — Crear conversación
- `GET /api/conversations/:id/messages` — Mensajes (DESC → normalize en cliente)
- `POST /api/conversations/:id/messages` — Enviar mensaje
- `GET /api/conversations/:id/messages/search` — Búsqueda full-text
- `GET /api/call-logs` — Historial de llamadas
- `GET /api/ws?token=...` — WebSocket (signaling + presencia)

## WebSocket — Signaling

Mensajes intercambiados para llamadas WebRTC:

| Tipo | Dirección | Descripción |
|------|-----------|-------------|
| `call-offer` | caller → callee | Inicia llamada con SDP offer |
| `call-answer` | callee → caller | Responde con SDP answer |
| `ice-candidate` | bidireccional | Intercambio de ICE candidates |
| `call-reject` | callee → caller | Rechaza llamada entrante |
| `call-end` | bidireccional | Finaliza llamada activa |
| `call-busy` | server → caller | Destino ocupado |
| `call-timeout` | server → caller | No contestó en 30s |
| `call-started` | server → ambos | Confirmación de conexión |
