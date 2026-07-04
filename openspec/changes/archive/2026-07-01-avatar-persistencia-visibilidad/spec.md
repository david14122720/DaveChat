# Delta: Avatar Persistence & Visibility

## Domain: Infrastructure

### ADDED Requirements

#### Requirement: Persistent Volume for Uploads

The Dokploy application MUST have a named persistent volume mounted at `/app/uploads`. The volume MUST survive container recreation and redeployment. Volume configuration MUST precede redeployment — never the reverse — to prevent data loss of existing uploads.

##### Scenario: Container redeployed

- GIVEN a persistent volume is mounted at `/app/uploads`
- WHEN the application is redeployed
- THEN existing uploads remain accessible via `/uploads/*`

##### Scenario: New upload after volume configured

- GIVEN the volume is mounted at `/app/uploads`
- WHEN a user uploads an avatar
- THEN the file is stored on the persistent volume
- AND it survives subsequent container recreation

#### Requirement: Volume Documentation

The project's `dokploy.md` MUST document the volume creation steps in the Dokploy UI, the mount path (`/app/uploads`), the required deployment sequence (configure volume first, then redeploy), and data persistence expectations.

##### Scenario: Volume setup documented

- GIVEN a developer reads `dokploy.md`
- WHEN they follow the Volume Setup section
- THEN they can configure the volume correctly
- AND they understand the configure-before-redeploy sequence

## Domain: User Profiles

### MODIFIED Requirements

#### Requirement: Client-Side Image Upload

The client MUST provide a clickable avatar that opens a file picker (`accept: image/*`). On selection, the image MUST be loaded onto a canvas, resized to 400×400 (maintaining aspect ratio), converted to Blob (JPEG), and submitted via `FormData` to the upload endpoint. Avatar display in ALL surfaces — profile page, ChatWindow header, and contact list sidebar — MUST render an `<img>` when `avatar_url` is set and an initials-in-circle fallback when null. The contact list MUST use the same dimensions (`w-10 h-10`) and style (`rounded-full`, `object-cover`) as the existing ChatWindow pattern to avoid layout shifts.
(Previously: avatar display specified generically without mentioning contact list sidebar)

##### Scenario: Contact with avatar

- GIVEN a contact has `avatar_url` set to a valid URL
- WHEN the sidebar contact list renders
- THEN the contact shows an `<img>` tag with that URL at `w-10 h-10 rounded-full object-cover`

##### Scenario: Contact without avatar

- GIVEN a contact has `avatar_url` set to null
- WHEN the sidebar contact list renders
- THEN the contact shows the initials-in-circle fallback (existing behavior, unchanged)

##### Scenario: ChatWindow unaffected

- GIVEN ChatWindow already renders avatars
- WHEN this change is deployed
- THEN ChatWindow avatar display is unchanged
