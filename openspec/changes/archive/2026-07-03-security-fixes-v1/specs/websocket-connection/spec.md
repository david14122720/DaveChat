# Delta for websocket-connection

Behavior documented in `openspec/specs/signaling/spec.md` under "Requirement: WebSocket Connection".

## ADDED Requirements

### Requirement: WebSocket Origin Validation

The server MUST validate the `Origin` header against a configured list of allowed origins before upgrading a WebSocket connection.

#### Scenario: Allowed origin upgrades successfully

- GIVEN the server is configured with `AllowedOrigins` containing `https://davechat.example.com`
- WHEN a WebSocket connection request has `Origin: https://davechat.example.com`
- THEN the HTTP connection upgrades to WebSocket (HTTP 101)

#### Scenario: Disallowed origin is rejected

- GIVEN the server is configured with `AllowedOrigins` containing `https://davechat.example.com`
- WHEN a WebSocket connection request has `Origin: https://evil.com`
- THEN the connection is rejected before upgrade with HTTP 403

#### Scenario: Missing origin rejected

- GIVEN the server has non-empty `AllowedOrigins`
- WHEN a WebSocket connection request has no `Origin` header
- THEN the connection is rejected before upgrade with HTTP 403

### Requirement: Client Sub-Protocol Token

The client MUST provide the JWT as the second argument to the WebSocket constructor, setting the `Sec-WebSocket-Protocol` header on the upgrade request.

#### Scenario: Token sent via sub-protocol

- GIVEN a valid JWT for user Alice
- WHEN `new WebSocket("ws://host/ws", [token])` is created
- THEN the upgrade request includes `Sec-WebSocket-Protocol: <token>`
- AND the connection upgrades on successful validation

## MODIFIED Requirements

### Requirement: WebSocket Connection

The system MUST expose `GET /ws` with JWT authentication. The JWT MUST be provided via the `Sec-WebSocket-Protocol` header (sub-protocol) during the upgrade handshake. The server MUST NOT accept JWT via the `?token=` query parameter.
(Previously: JWT was passed via query parameter `?token=<jwt>`)

#### Scenario: Valid token via sub-protocol upgrades to WebSocket

- GIVEN a valid JWT for user Alice
- WHEN Alice connects to `ws://host/ws` with `Sec-WebSocket-Protocol: <jwt>`
- THEN the HTTP connection upgrades to WebSocket (HTTP 101)
- AND Alice is registered in the hub as online

#### Scenario: Missing or invalid token is rejected

- GIVEN no token or an expired JWT
- WHEN connecting to `/ws` without sub-protocol header
- THEN the connection is rejected before upgrade with HTTP 403

#### Scenario: Token via query param is rejected

- GIVEN a valid JWT
- WHEN connecting to `ws://host/ws?token=<jwt>` without sub-protocol header
- THEN the connection is rejected before upgrade with HTTP 403

## REMOVED Requirements

### Requirement: URL Query Param Auth (from signaling spec)

The system no longer accepts `?token=<jwt>` query parameter for WebSocket authentication.
(Reason: Query parameters are logged by proxies and servers, leaking JWT tokens)
(Migration: Clients must use `new WebSocket(url, [token])` with the sub-protocol pattern)
