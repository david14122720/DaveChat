# API Gateway Specification

## Purpose

Go HTTP router exposing all REST endpoints and WebSocket on a single port (8080), with CORS, logging, and JWT middleware.

## Route Table

| Method | Path | Auth | Handler |
|--------|------|------|---------|
| `POST` | `/api/auth/register` | No | Register |
| `POST` | `/api/auth/login` | No | Login |
| `GET` | `/api/profiles/me` | Yes | GetOwnProfile |
| `PATCH` | `/api/profiles/me` | Yes | UpdateProfile |
| `GET` | `/api/profiles` | Yes | ListProfiles |
| `POST` | `/api/conversations` | Yes | CreateConversation |
| `GET` | `/api/conversations` | Yes | ListConversations |
| `POST` | `/api/conversations/{id}/messages` | Yes | SendMessage |
| `GET` | `/api/conversations/{id}/messages` | Yes | ListMessages |
| `GET` | `/ws` | Yes (query) | WebSocket upgrade |
| `GET` | `/health` | No | Health check |

## Requirements

### Requirement: CORS for Dev Mode

The system MUST allow cross-origin requests from `http://localhost:5173` (Vite dev server).

#### Scenario: Preflight OPTIONS succeeds

- GIVEN a request with `Origin: http://localhost:5173`
- WHEN the browser sends an OPTIONS preflight
- THEN response includes `Access-Control-Allow-Origin: http://localhost:5173`
- AND `Access-Control-Allow-Methods: GET, POST, PATCH, DELETE, OPTIONS`

### Requirement: Unified Port

The system MUST serve REST API and WebSocket on the same port (8080) using `gorilla/mux` router.

#### Scenario: REST and WS on same port

- GIVEN the Go server is running on port 8080
- WHEN a REST call is made to `http://localhost:8080/api/profiles`
- THEN it is routed to the REST handler
- WHEN a WebSocket connection is made to `ws://localhost:8080/ws`
- THEN it is routed to the WebSocket handler

### Requirement: Frontend api.js Client

The system MUST provide a fetch/axios wrapper `client/src/api.js` that injects the JWT from localStorage.

#### Scenario: Authenticated request includes token

- GIVEN a JWT stored in `localStorage` as `davechat_token`
- WHEN the frontend calls `api.get("/profiles")`
- THEN the request includes `Authorization: Bearer <token>`

#### Scenario: 401 response clears token

- GIVEN the API returns 401
- WHEN the fetch wrapper receives the response
- THEN the token is removed from localStorage
- AND the user is redirected to the login screen
