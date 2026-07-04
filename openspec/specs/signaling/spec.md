# Signaling Specification

## Purpose

WebSocket hub for relaying real-time events (WebRTC signaling: offer, answer, ICE candidates; call state: reject, busy, end; presence).

## Requirements

> **NOTE**: WebSocket connection behavior (auth, upgrade, origin validation) has been moved to its own specification at `openspec/specs/websocket-connection/spec.md` as part of the security-fixes-v1 change. The signaling relay semantics below are unchanged.

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
