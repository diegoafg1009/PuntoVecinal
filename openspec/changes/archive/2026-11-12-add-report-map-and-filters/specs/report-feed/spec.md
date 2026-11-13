# Spec Delta

## MODIFIED Requirements

### Requirement: The API lists published reports from newest to oldest
The API SHALL return, for a caller with a valid session, the published reports ordered from the most recent to the oldest publication date. The pilot has a single zone, so the list SHALL include every report, without a zone filter, unless the caller filters by category or status as `report-filters` describes.

#### Scenario: Newest first
- **GIVEN** reports published at 10:00, 11:00 and 12:00
- **WHEN** a caller with a valid session lists the reports
- **THEN** the list starts with the 12:00 report, then 11:00, then 10:00

#### Scenario: Anonymous session
- **GIVEN** a caller with an anonymous session
- **WHEN** they list the reports
- **THEN** the API answers 200 with the list

#### Scenario: No reports yet
- **WHEN** no report has been published and a caller lists the reports
- **THEN** the API answers 200 with an empty list and no next page

#### Scenario: No filter
- **GIVEN** reports in several categories and statuses
- **WHEN** a caller lists the reports without a category or status filter
- **THEN** every published report is listed

#### Scenario: No token
- **WHEN** the request carries no valid token
- **THEN** the API answers 401

### Requirement: Inicio shows loading, empty and error states
Inicio SHALL show a loading state while the first page arrives. When there are no reports yet and no filter is on, it SHALL say so and offer «Reportar». When the first page cannot be loaded, it SHALL show a message in Spanish with «Reintentar». With filters on and no match, it SHALL show the empty state that `report-filters` describes.

#### Scenario: Loading
- **WHEN** Inicio opens and the first page has not arrived yet
- **THEN** a loading state is shown in place of the cards

#### Scenario: No reports
- **WHEN** the list is empty and no filter is on
- **THEN** Inicio says there are no reports yet and offers «Reportar»

#### Scenario: Load failure and retry
- **GIVEN** the API cannot be reached when Inicio opens
- **THEN** a message in Spanish says the reports could not be loaded and offers «Reintentar»
- **WHEN** the neighbor taps «Reintentar» and the API answers
- **THEN** the cards are shown
