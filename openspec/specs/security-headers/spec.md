# Security Headers Specification

## Purpose

Echo middleware that sets security response headers (CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy) on every HTTP response. All headers MUST be gated behind the server's DevMode flag — they MUST NOT be applied in development mode.

## Requirements

### Requirement: DevMode Gate

The security headers middleware MUST only apply when the server's configuration has `DevMode: false`. When `DevMode: true`, no security headers MUST be set.

#### Scenario: Headers present in production

- GIVEN the server is running with `DevMode: false`
- WHEN any HTTP request is made
- THEN the response MUST include all security headers

#### Scenario: Headers omitted in DevMode

- GIVEN the server is running with `DevMode: true`
- WHEN any HTTP request is made
- THEN the response MUST NOT include any of the security headers

### Requirement: Content-Security-Policy

The system SHOULD start with `Content-Security-Policy-Report-Only` during the initial tuning phase before enforcing a `Content-Security-Policy`.

#### Scenario: CSP Report-Only in production

- GIVEN the server is in production mode
- WHEN any HTTP response is served
- THEN the `Content-Security-Policy-Report-Only` header is present

### Requirement: HSTS

The system MUST set `Strict-Transport-Security: max-age=31536000; includeSubDomains`.

#### Scenario: HSTS header present

- GIVEN the server is in production mode
- WHEN any HTTP or HTTPS response is served
- THEN `Strict-Transport-Security` is `max-age=31536000; includeSubDomains`

### Requirement: Frame Options

The system MUST set `X-Frame-Options: DENY`.

#### Scenario: X-Frame-Options header

- GIVEN the server is in production mode
- WHEN any HTTP response is served
- THEN `X-Frame-Options` is `DENY`

### Requirement: Content Type Options

The system MUST set `X-Content-Type-Options: nosniff`.

#### Scenario: X-Content-Type-Options header

- GIVEN the server is in production mode
- WHEN any HTTP response is served
- THEN `X-Content-Type-Options` is `nosniff`

### Requirement: Referrer Policy

The system MUST set `Referrer-Policy: strict-origin-when-cross-origin`.

#### Scenario: Referrer-Policy header

- GIVEN the server is in production mode
- WHEN any HTTP response is served
- THEN `Referrer-Policy` is `strict-origin-when-cross-origin`

### Requirement: Permissions Policy

The system MUST set a `Permissions-Policy` header with minimal defaults disabling unused browser features.

#### Scenario: Permissions-Policy header

- GIVEN the server is in production mode
- WHEN any HTTP response is served
- THEN `Permissions-Policy` restricts sensitive API access (camera, microphone, geolocation) to same-origin only
