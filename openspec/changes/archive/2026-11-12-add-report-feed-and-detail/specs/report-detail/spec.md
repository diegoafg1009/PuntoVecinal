# Spec Delta

## ADDED Requirements

### Requirement: The report page shows the report
The report page SHALL show the report's photos, category, status, title, author, publication date, place reference, details and the place on a map, for any neighbor with or without an account. Parts the report does not have (details, reference) SHALL NOT be shown empty.

#### Scenario: Open a report
- **GIVEN** a report in Limpieza, «Publicado», published 2 hours ago by «Rosa», with the reference «Jr. Ejemplo cdra. 3», details and 2 photos
- **WHEN** a neighbor opens its page
- **THEN** they see its photos, «Limpieza», «Publicado», the title, «Rosa · hace 2 h · Jr. Ejemplo cdra. 3», the details and a map with a pin at the report's location

#### Scenario: Without optional parts
- **GIVEN** a report without details and without a reference
- **WHEN** a neighbor opens its page
- **THEN** no empty details nor empty reference are shown

#### Scenario: Status labels
- **WHEN** the page shows the status
- **THEN** it uses the same Spanish labels as the cards of Inicio

### Requirement: A report without an author name shows «Vecino anónimo»
The report page SHALL show «Vecino anónimo» as the author of a report that has no author name.

#### Scenario: Anonymous author
- **GIVEN** a report published without an author name
- **WHEN** a neighbor opens its page
- **THEN** the author shown is «Vecino anónimo»

### Requirement: Publication date is relative when recent
The report page and the cards of Inicio SHALL show the publication date as time elapsed when the report is less than 7 days old («hace un momento», «hace 5 min», «hace 2 h», «ayer», «hace 3 días»), and as a date («12 nov 2026») when it is older.

#### Scenario: Recent report
- **WHEN** a report was published 2 hours ago
- **THEN** its date is shown as «hace 2 h»

#### Scenario: Old report
- **WHEN** a report was published 10 days ago
- **THEN** its date is shown as the day, month and year

### Requirement: Photos are browsed one at a time
The report page SHALL show one photo at a time, let the neighbor move to the others, and show which one they see out of how many («1 / 2») when there is more than one. A photo that cannot be loaded SHALL be replaced by a placeholder.

#### Scenario: Several photos
- **GIVEN** a report with 2 photos
- **WHEN** a neighbor opens its page
- **THEN** they see the first photo with «1 / 2»
- **WHEN** they move to the next photo
- **THEN** they see the second photo with «2 / 2»

#### Scenario: One photo
- **GIVEN** a report with 1 photo
- **WHEN** a neighbor opens its page
- **THEN** the photo is shown without a counter

#### Scenario: Photo cannot be loaded
- **WHEN** a photo cannot be loaded
- **THEN** a placeholder is shown in its place and the rest of the page stays as it is

### Requirement: The report page handles a missing report and load failures
The report page SHALL show a loading state until the report arrives, a message in Spanish with a way back to Inicio when the report does not exist, and a message in Spanish with «Reintentar» when it cannot be loaded.

#### Scenario: Loading
- **WHEN** the page opens and the report has not arrived yet
- **THEN** a loading state is shown

#### Scenario: Report does not exist
- **WHEN** the neighbor opens the page of a report that does not exist
- **THEN** a message in Spanish says the report was not found and offers to go to Inicio

#### Scenario: Load failure and retry
- **GIVEN** the API cannot be reached when the page opens
- **THEN** a message in Spanish says the report could not be loaded and offers «Reintentar»
- **WHEN** the neighbor taps «Reintentar» and the API answers
- **THEN** the report is shown

### Requirement: The report page has its own top bar on mobile
On mobile, the report page SHALL show a top bar with a back button and the title «Reporte» in place of the app top bar, and the bottom bar SHALL stay fixed while the neighbor scrolls. The back button SHALL return to the previous screen of the app, or to Inicio when the page was opened directly.

#### Scenario: Scroll on mobile
- **GIVEN** a neighbor on a mobile device on a report page longer than the screen
- **WHEN** they scroll
- **THEN** the bottom bar stays fixed at the bottom of the screen

#### Scenario: Back from the feed
- **GIVEN** the neighbor opened the report from a card of Inicio
- **WHEN** they tap the back button
- **THEN** Inicio is shown again

#### Scenario: Back after opening the link directly
- **GIVEN** the neighbor opened the report page from a link, with no previous screen in the app
- **WHEN** they tap the back button
- **THEN** Inicio opens

#### Scenario: Desktop
- **GIVEN** the neighbor is on desktop
- **WHEN** they open a report page
- **THEN** the app header stays and the report is shown centered, with the same content and order as on mobile
