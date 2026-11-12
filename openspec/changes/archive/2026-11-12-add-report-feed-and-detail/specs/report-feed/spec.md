# Spec Delta

## Purpose

Defines how neighbors, with or without an account, see the published reports as a list: the paginated API operation that returns them and the Inicio screen that shows them as cards, page by page, with the distance to each report when the neighbor shares their location.

## ADDED Requirements

### Requirement: The API lists published reports from newest to oldest
The API SHALL return, for a caller with a valid session, the published reports ordered from the most recent to the oldest publication date. The pilot has a single zone, so the list SHALL include every report, without a zone filter.

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

#### Scenario: No token
- **WHEN** the request carries no valid token
- **THEN** the API answers 401

### Requirement: Each listed report carries what a card shows
Each item of the list SHALL carry the report id, title, category (slug and label), status, place reference (null when there is none), publication date and the link to its first photo, which expires after one hour. Items SHALL NOT carry the details, the author name nor any user id.

#### Scenario: Item contents
- **GIVEN** a report with 2 photos and the reference «frente al mercado»
- **WHEN** it is listed
- **THEN** its item carries its id, title, category, status, the reference, its publication date and one link that shows its first photo

#### Scenario: The publisher is never revealed
- **WHEN** any caller lists the reports
- **THEN** no item carries an author name nor the user id of the session that published it

### Requirement: The list is paginated with a cursor
The API SHALL return the list in pages of at most the requested size (20 by default, between 1 and 50), with an opaque cursor for the next page, or none when there are no more. A next page SHALL continue right after the last item of the previous one, without repeating or skipping reports, even when new reports were published in between.

#### Scenario: Two pages
- **GIVEN** 25 published reports
- **WHEN** a caller asks for the first page without a size and then for the page of the cursor it returned
- **THEN** the first page has the 20 newest reports and a cursor
- **AND** the second page has the other 5 and no cursor

#### Scenario: New report between pages
- **GIVEN** a caller got the first page and its cursor
- **WHEN** a new report is published and the caller asks for the next page
- **THEN** the next page does not repeat any report of the first page nor skip any older report

#### Scenario: Same publication time
- **GIVEN** two reports with the same publication date that fall on both sides of a page boundary
- **WHEN** the caller walks through the pages
- **THEN** each of them appears exactly once

#### Scenario: Invalid cursor or size
- **WHEN** the cursor is not one the API issued, or the size is below 1 or above 50
- **THEN** the API answers 400

### Requirement: Distance is given only when the caller sends a location
The API SHALL accept an optional caller location (latitude and longitude, both or neither) and, when it is given, SHALL return for each item its distance in whole meters from that location. Without a location, items SHALL carry no distance. The order SHALL NOT depend on the location.

#### Scenario: With location
- **GIVEN** a report about 120 m from the caller
- **WHEN** the caller lists the reports sending their latitude and longitude
- **THEN** that item carries a distance of about 120 meters

#### Scenario: Without location
- **WHEN** the caller lists the reports without a location
- **THEN** no item carries a distance

#### Scenario: Incomplete or invalid location
- **WHEN** only one of latitude and longitude is sent, or one is out of range
- **THEN** the API answers 400

### Requirement: Inicio shows the reports as cards
Inicio SHALL show the published reports from newest to oldest as cards. Each card SHALL show the first photo, the title, the category, the status, the place reference when there is one, the distance when known and how long ago the report was published.

#### Scenario: Open Inicio
- **WHEN** a neighbor, with or without an account, opens Inicio
- **THEN** they see the reports as cards, the most recent first

#### Scenario: Card contents
- **GIVEN** a report «Basura acumulada en la esquina del parque» in Limpieza, «Publicado», published 2 hours ago, with the reference «Jr. Ejemplo cdra. 3»
- **WHEN** its card is shown without the neighbor's location
- **THEN** it shows its photo, the title, «Limpieza», «Publicado», «Jr. Ejemplo cdra. 3» and «hace 2 h»

#### Scenario: Status labels
- **WHEN** a card shows a status
- **THEN** it uses the Spanish label of the status: «Publicado», «Derivado», «Recibido por la entidad», «En atención», «Atendido según la entidad» or «Cerrado»

#### Scenario: Photo cannot be shown
- **WHEN** the photo of a card cannot be loaded
- **THEN** the card shows a placeholder in its place and the rest of the card stays as it is

### Requirement: Inicio loads more reports while scrolling
Inicio SHALL load the next page of reports when the neighbor reaches the end of the list, without reloading the page, and SHALL stop when there are no more.

#### Scenario: Reach the end
- **GIVEN** there are more reports than fit in the first page
- **WHEN** the neighbor scrolls to the end of the list
- **THEN** the next reports appear below the ones already shown, without reloading the page

#### Scenario: Nothing more
- **GIVEN** all reports are already shown
- **WHEN** the neighbor reaches the end of the list
- **THEN** no more reports are requested

#### Scenario: Next page fails
- **GIVEN** the neighbor reached the end of the list and the next page cannot be loaded
- **THEN** the reports already shown stay and a message in Spanish offers «Reintentar»

### Requirement: Inicio shows loading, empty and error states
Inicio SHALL show a loading state while the first page arrives, a message when there are no reports yet, and a message in Spanish with «Reintentar» when the first page cannot be loaded.

#### Scenario: Loading
- **WHEN** Inicio opens and the first page has not arrived yet
- **THEN** a loading state is shown in place of the cards

#### Scenario: No reports
- **WHEN** the list is empty
- **THEN** Inicio says there are no reports yet and offers «Reportar»

#### Scenario: Load failure and retry
- **GIVEN** the API cannot be reached when Inicio opens
- **THEN** a message in Spanish says the reports could not be loaded and offers «Reintentar»
- **WHEN** the neighbor taps «Reintentar» and the API answers
- **THEN** the cards are shown

### Requirement: Distance appears only when the neighbor shares their location
Inicio SHALL offer a «Usar mi ubicación» button and SHALL ask for the device location only when the neighbor taps it. Once shared, cards SHALL show the distance («120 m», «1,2 km») and the app SHALL use the location again on later visits from that device while the browser keeps the permission, without asking again. Without it, cards SHALL NOT show a distance.

#### Scenario: Nothing is asked on its own
- **WHEN** a neighbor opens Inicio for the first time
- **THEN** the app does not ask for the device location
- **AND** the cards show no distance

#### Scenario: Share the location
- **WHEN** the neighbor taps «Usar mi ubicación» and accepts
- **THEN** the list is loaded again and each card shows its distance

#### Scenario: Distance format
- **WHEN** a report is 123 m away, and another one 1234 m away
- **THEN** their cards show «120 m» and «1,2 km»

#### Scenario: Decline or failure
- **WHEN** the neighbor taps «Usar mi ubicación» and declines, or the position cannot be obtained
- **THEN** a message in Spanish says the distance could not be shown
- **AND** the cards stay without distance

#### Scenario: Later visit
- **GIVEN** the neighbor shared their location on a previous visit and the browser still allows it
- **WHEN** they open Inicio again
- **THEN** the cards show the distance without asking again

### Requirement: A card opens the report
Tapping a card SHALL open the page of that report.

#### Scenario: Open a report
- **WHEN** the neighbor taps the card of a report
- **THEN** the page of that report opens
