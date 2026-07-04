# Infrastructure Specification

## Purpose

Docker and Dokploy infrastructure configuration for the DaveChat application, including persistent volume management for uploaded files and application deployment.

## Requirements

### Requirement: Persistent Volume for Uploads

The Dokploy application MUST have a named persistent volume mounted at `/app/uploads`. The volume MUST survive container recreation and redeployment. Volume configuration MUST precede redeployment — never the reverse — to prevent data loss of existing uploads.

#### Scenario: Container redeployed

- GIVEN a persistent volume is mounted at `/app/uploads`
- WHEN the application is redeployed
- THEN existing uploads remain accessible via `/uploads/*`

#### Scenario: New upload after volume configured

- GIVEN the volume is mounted at `/app/uploads`
- WHEN a user uploads an avatar
- THEN the file is stored on the persistent volume
- AND it survives subsequent container recreation

### Requirement: Volume Documentation

The project's `dokploy.md` MUST document the volume creation steps in the Dokploy UI, the mount path (`/app/uploads`), the required deployment sequence (configure volume first, then redeploy), and data persistence expectations.

#### Scenario: Volume setup documented

- GIVEN a developer reads `dokploy.md`
- WHEN they follow the Volume Setup section
- THEN they can configure the volume correctly
- AND they understand the configure-before-redeploy sequence
