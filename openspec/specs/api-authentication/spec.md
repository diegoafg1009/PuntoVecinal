# api-authentication Specification

## Purpose

Defines how the API identifies its callers: only valid Supabase sessions, including anonymous ones, may use protected endpoints, and a caller can ask who the API believes they are. Sign-in, sign-up and token refresh are provided by Supabase, not by the API.

## Requirements

### Requirement: Protected endpoints require a valid session
The API SHALL reject with 401 any request to a protected endpoint that has no `Authorization: Bearer` token, or whose token is malformed, expired, has an invalid signature, or was not issued by the configured Supabase project.

#### Scenario: Missing token
- **WHEN** a request without `Authorization` is sent to a protected endpoint
- **THEN** the API responds 401

#### Scenario: Invalid or expired token
- **WHEN** a request carries an expired token or one with an invalid signature
- **THEN** the API responds 401 and does not run the endpoint logic

### Requirement: Anonymous sessions are accepted
The API SHALL accept a valid Supabase token of an anonymous session on protected endpoints, with the same validation as for a user with an account.

#### Scenario: Valid anonymous token
- **WHEN** a request carries a valid token of an anonymous session
- **THEN** the API processes the request as that user

### Requirement: Caller identity endpoint
The API SHALL expose `GET /me`, which returns the authenticated caller's user id and whether the session is anonymous.

#### Scenario: Anonymous caller
- **WHEN** a caller with a valid anonymous token requests `GET /me`
- **THEN** the API responds 200 with the user id from the token and `isAnonymous` set to true

#### Scenario: Same user across requests
- **WHEN** the same anonymous token is used on two requests to `GET /me`
- **THEN** both responses return the same user id

### Requirement: Authentication is delegated to Supabase
The API SHALL NOT offer endpoints to sign in, register or refresh a token (`GET /me` only reports identity); clients obtain and refresh sessions directly from Supabase, and the API accepts the resulting tokens.

#### Scenario: Refreshed token
- **WHEN** a client refreshes its session with Supabase and sends the new token to `GET /me`
- **THEN** the API accepts it and returns the same user id

### Requirement: Public endpoints stay public
The API SHALL keep `GET /healthz` reachable without a token.

#### Scenario: Health check without token
- **WHEN** `GET /healthz` is requested without `Authorization`
- **THEN** the API does not respond 401
