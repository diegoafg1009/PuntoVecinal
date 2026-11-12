# Spec Delta

## Purpose

Defines what the API does when a neighbor publishes a report: it validates the report, cleans and stores its photos, records who published it without exposing them, and limits abuse.

## ADDED Requirements

### Requirement: A report is published with photos and place
The API SHALL let a caller with a valid session publish a report by sending a category, a title, an optional details text, a location, an optional place reference and 1 to 3 photos in a single request, and SHALL answer with the id of the created report and the status «Publicado».

#### Scenario: Publish a complete report
- **GIVEN** a caller with a valid session
- **WHEN** they publish a report with a known category, a title, a location and 2 photos
- **THEN** the report is created with status «Publicado»
- **AND** the response carries the report id

#### Scenario: Optional fields left out
- **WHEN** a caller publishes a report without details and without a place reference
- **THEN** the report is created

#### Scenario: No token
- **WHEN** the request carries no valid token
- **THEN** the API answers 401 and creates nothing

### Requirement: Published data is validated
The API SHALL reject, with a 400 and without creating anything, a report whose category is not one of the fixed categories, whose title is empty or only whitespace or longer than 100 characters, whose details exceed 1000 characters, whose place reference exceeds 150 characters, whose latitude or longitude is missing or out of range, or whose photo count is not between 1 and 3.

#### Scenario: Unknown category
- **WHEN** a report is published with a category that does not exist
- **THEN** the API answers 400 and creates nothing

#### Scenario: Blank title
- **WHEN** the title contains only spaces
- **THEN** the API answers 400 and creates nothing

#### Scenario: No photos
- **WHEN** a report is published without photos
- **THEN** the API answers 400 and creates nothing

#### Scenario: Too many photos
- **WHEN** a report is published with 4 photos
- **THEN** the API answers 400 and creates nothing

#### Scenario: Location out of range
- **WHEN** the latitude is above 90 or the longitude is above 180 in absolute value
- **THEN** the API answers 400 and creates nothing

### Requirement: Photos are cleaned and stored by the server
The API SHALL accept only photos that can be read as JPEG or PNG images within a maximum size, SHALL re-encode each one so that the stored file carries no location nor other hidden file data, and SHALL store them in the object storage, ordered as received.

#### Scenario: Photo with device location
- **WHEN** a report is published with a photo whose file carries the device location
- **THEN** the stored photo carries no location nor other hidden file data

#### Scenario: File that is not an image
- **WHEN** one of the photos cannot be read as an image
- **THEN** the API answers 400 and creates nothing

#### Scenario: Photo too large
- **WHEN** a photo exceeds the maximum size
- **THEN** the API answers 413 and creates nothing

#### Scenario: Photo order
- **WHEN** a report is published with 3 photos
- **THEN** the stored photos keep the order in which they were sent

### Requirement: Nothing is left half-published
The API SHALL either create the report with all its photos or create nothing visible, and SHALL NOT leave stored photo files that belong to no report.

#### Scenario: Storage fails
- **WHEN** a photo cannot be stored
- **THEN** the API answers with a server error and no report exists

#### Scenario: Database fails after the photos were stored
- **WHEN** the report cannot be saved after its photos were stored
- **THEN** the API answers with a server error, no report exists and the stored photos are removed

### Requirement: Author name is optional
The API SHALL accept an optional author name of up to 40 characters, trimmed, and SHALL store it as the public author of the report only when it is not empty. A report without an author name is public as «Vecino anónimo».

#### Scenario: Named report
- **WHEN** a report is published with the author name «Rosa»
- **THEN** the report's public author is «Rosa»

#### Scenario: Anonymous report
- **WHEN** a report is published without an author name, or with only spaces
- **THEN** the report has no public author name

#### Scenario: Name too long
- **WHEN** the author name has more than 40 characters
- **THEN** the API answers 400 and creates nothing

### Requirement: The publishing session is recorded privately
The API SHALL record the user id of the session that published each report, whether the session is anonymous or registered, SHALL use it to tie the report to that neighbor, and SHALL NOT reveal it in any response.

#### Scenario: Anonymous report is tied to the session
- **GIVEN** a caller with an anonymous session
- **WHEN** they publish a report without an author name
- **THEN** the report is stored with that session's user id
- **AND** the response does not include the user id

### Requirement: Publication rate is limited
The API SHALL allow one neighbor to publish at most 5 reports in any 10-minute window and SHALL answer 429 with the time to wait, creating nothing, when the limit is exceeded.

#### Scenario: Limit exceeded
- **GIVEN** a neighbor who published 5 reports in the last 10 minutes
- **WHEN** they publish another one
- **THEN** the API answers 429 with how many seconds to wait
- **AND** nothing is created

#### Scenario: Limit is per neighbor
- **GIVEN** one neighbor reached the limit
- **WHEN** a different neighbor publishes a report
- **THEN** it is created

#### Scenario: Window passes
- **GIVEN** a neighbor reached the limit
- **WHEN** the oldest of those reports is more than 10 minutes old and they publish again
- **THEN** the report is created
