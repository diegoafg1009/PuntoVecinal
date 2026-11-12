# report-detail Specification

## Purpose

Defines how the API returns one report to a caller, including its photos through temporary links. The report page that shows it belongs to HU-02 (PV-11).

## Requirements

### Requirement: The API returns one report
The API SHALL return, for a caller with a valid session, a report by its id with its title, details, category, status, location, place reference, publication date, author name (empty when the report has none) and its photos in the order they were published, and SHALL answer 404 when the report does not exist.

#### Scenario: Existing report
- **GIVEN** a published report
- **WHEN** a caller with a valid session asks for it by id
- **THEN** the API returns its title, details, category, status «publicado», location, reference, date, author name and photos

#### Scenario: Unknown report
- **WHEN** the id does not match any report, or is not a valid id
- **THEN** the API answers 404

#### Scenario: No token
- **WHEN** the request carries no valid token
- **THEN** the API answers 401

#### Scenario: The publisher is never revealed
- **WHEN** any caller asks for a report published without an author name
- **THEN** the response carries no author name and no user id, whichever session published it

### Requirement: Photos are served through temporary links
The API SHALL return each photo as a temporary signed link that expires after one hour, and SHALL NOT expose the storage bucket publicly.

#### Scenario: Photo links
- **WHEN** a report with 2 photos is returned
- **THEN** it carries 2 links, in order, that show the photos while they are valid

#### Scenario: Expired link
- **WHEN** a link is used after it expired
- **THEN** the storage refuses it
