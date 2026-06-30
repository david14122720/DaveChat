# Signaling Specification

## Purpose

WebSocket hub for relaying real-time events (WebRTC signaling: offer, answer, ICE candidates; call state: reject, busy, end; presence).

## Requirements

### Requirement: WebSocket Connection

The system MUST expose `GET /ws` with JWT auth via `?token=<jwt>` query parameter.

#### Scenario: Valid token upgrades to WebSocket

- GIVEN a valid JWT for user Alice
- WHEN Alice connects to `ws://host/ws?token=<jwt>`
- THEN the HTTP connection upgrades to WebSocket (HTTP 101)
- AND Alice is registered in the hub as online

#### Scenario: Missing or invalid token is rejected

- GIVEN no token or an expired JWT
- WHEN connecting to `/ws`
- THEN the connection is rejected with HTTP 401

### Requirement: Event Relay

The system MUST relay signaling messages to the target user's WebSocket connection.

#### Scenario: Offer reaches target

- GIVEN Alice and Bob are both connected via WebSocket
- WHEN Alice sends `{type: "signal:offer", target: "<bob-id>", sdp: "..."}`
- THEN Bob receives `{type: "signal:offer", from: "<alice-id>", sdp: "..."}`

#### Scenario: Offline target is notified

- GIVEN Alice is connected, Bob is offline
- WHEN Alice sends a signal targeted at Bob
- THEN Alice receives `{type: "error", message: "user offline"}`

### Requirement: Presence Broadcast

The system MUST broadcast `user:online` and `user:offline` events to all connected clients.

#### Scenario: Online notification

- GIVEN Bob is connected
- WHEN Alice establishes a WebSocket connection
- THEN Bob receives `{type: "user:online", userId: "<alice-id>"}`
