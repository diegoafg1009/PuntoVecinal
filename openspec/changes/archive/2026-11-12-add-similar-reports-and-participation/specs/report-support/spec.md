# Spec Delta

## Purpose

Defines «También me afecta» (HU-15, PV-24): how a neighbor marks that a report's problem also affects them, who counts as affected, and how the report page shows it, so it is visible how many people suffer each problem.

## ADDED Requirements

### Requirement: A neighbor marks and unmarks support
The API SHALL let a caller with a valid session mark «También me afecta» on a report and remove it again. Marking twice SHALL count once, and removing support that was not marked SHALL change nothing. Both SHALL answer with the report's affected count and the caller's support state.

#### Scenario: Mark
- **GIVEN** a report that affects 3 neighbors
- **WHEN** a caller who has not supported it marks support
- **THEN** the API answers with 4 and the state «supporter»

#### Scenario: Mark twice
- **GIVEN** a caller who already supports a report that affects 4
- **WHEN** they mark support again
- **THEN** the API answers with 4 and the state «supporter»

#### Scenario: Unmark
- **GIVEN** a caller who supports a report that affects 4
- **WHEN** they remove their support
- **THEN** the API answers with 3 and the state «none»

#### Scenario: Unknown report
- **WHEN** a caller marks or removes support on a report that does not exist, or with an id that is not valid
- **THEN** the API answers 404

#### Scenario: No token
- **WHEN** the request carries no valid token
- **THEN** the API answers 401

### Requirement: The author always counts as affected
The affected count of a report SHALL be its author plus every other neighbor who supports it, so a new report affects 1. The author SHALL NOT be able to mark or remove support on their own report. The API SHALL answer 409 without changing anything when they try.

#### Scenario: New report
- **WHEN** a report has just been published
- **THEN** its affected count is 1

#### Scenario: Author marks support
- **WHEN** the session that published a report marks support on it
- **THEN** the API answers 409 and the count does not change

#### Scenario: Author removes support
- **WHEN** the session that published a report removes support on it
- **THEN** the API answers 409 and the count does not change

### Requirement: Support is one per session and stays private
Support SHALL be tied to the user id of the caller's session, anonymous or not, so one session counts at most once per report. The API SHALL NOT reveal who supports a report, only how many.

#### Scenario: Two neighbors
- **GIVEN** a report that affects 1
- **WHEN** two different sessions mark support
- **THEN** it affects 3

#### Scenario: Supporters are not listed
- **WHEN** any caller reads a report or marks support
- **THEN** no response carries the user ids of the supporters

### Requirement: Support does not change the status
Marking or removing support SHALL NOT change the status of the report.

#### Scenario: Status after support
- **GIVEN** a report in «Publicado»
- **WHEN** 10 neighbors mark support
- **THEN** its status is still «Publicado»

### Requirement: The report page shows «También me afecta»
The report page SHALL show a «También me afecta · N» button with the affected count, which looks different when the neighbor supports the report. Tapping it SHALL mark support when not marked and remove it when marked, and the count SHALL go up or down by one. For the author, the button SHALL show as marked and SHALL NOT be tappable.

#### Scenario: Mark from the page
- **GIVEN** a report that affects 14 and the neighbor has not supported it
- **WHEN** they tap «También me afecta · 14»
- **THEN** the button shows as marked with «También me afecta · 15»

#### Scenario: Unmark from the page
- **GIVEN** the neighbor supports a report that affects 15
- **WHEN** they tap the button
- **THEN** the button shows as not marked with «También me afecta · 14»

#### Scenario: Author
- **GIVEN** the neighbor opens a report they published
- **THEN** the button shows as marked and tapping it does nothing

#### Scenario: Request fails
- **GIVEN** the API cannot be reached
- **WHEN** the neighbor taps the button
- **THEN** the button and the count go back to what they were and a message in Spanish says the change could not be saved

#### Scenario: Quick taps
- **WHEN** the neighbor taps the button several times quickly
- **THEN** the final state shown matches the state saved by the API
