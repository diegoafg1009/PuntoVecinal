# Spec Delta

## Purpose

Defines the fixed categories a report can have and how the app learns them from the API, so that the form and the backend always agree on the same list.

## ADDED Requirements

### Requirement: Categories are fixed and stored by the system
The system SHALL keep exactly five report categories: Limpieza, Alumbrado, Pistas y veredas, Agua y desagüe and Otro. Each category SHALL have a stable `slug` (`limpieza`, `alumbrado`, `pistas-y-veredas`, `agua-y-desague`, `otro`), a Spanish `label` and a fixed position. No user role SHALL be able to add, rename or remove categories from the app.

#### Scenario: Categories exist on a fresh database
- **WHEN** the database is created from its migrations
- **THEN** the five categories exist with those slugs, labels and positions, without any extra seeding step

### Requirement: Listing categories
The API SHALL return the list of categories at `GET /categories`, ordered by position, each with its `slug` and `label`.

#### Scenario: Authenticated neighbor lists categories
- **GIVEN** a valid session, anonymous or with an account
- **WHEN** the client calls `GET /categories`
- **THEN** the response is 200 with the five categories in this order: Limpieza, Alumbrado, Pistas y veredas, Agua y desagüe, Otro
- **AND** each item has a `slug` and a `label`

#### Scenario: Same answer on every call
- **WHEN** the client calls `GET /categories` twice
- **THEN** both responses contain the same items in the same order

### Requirement: Listing categories needs a session
The API SHALL reject `GET /categories` without a valid session, as it does for every protected operation.

#### Scenario: No token
- **WHEN** the client calls `GET /categories` without a token or with an invalid one
- **THEN** the response is 401

### Requirement: Categories are unavailable when the database is
The API SHALL NOT answer `GET /categories` with an invented or partial list when the categories cannot be read.

#### Scenario: Database failure
- **GIVEN** the database cannot be queried
- **WHEN** the client calls `GET /categories`
- **THEN** the response is a server error, not a list
