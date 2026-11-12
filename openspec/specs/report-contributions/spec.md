# report-contributions Specification

## Purpose
Defines aportes (HU-16, PV-25): a text with an optional photo that any neighbor adds to an existing report to keep the case up to date, the API that stores and lists them, and the «Aportar» button and aportes list on the report page.

## Requirements

### Requirement: A neighbor adds an aporte to a report
The API SHALL let a caller with a valid session add an aporte to an existing report. The aporte is a text of 1 to 1200 characters (trimmed), optionally one photo and optionally an author name of up to 40 characters. The API SHALL answer with the created aporte. The report page reaches this through «Sumarte a este reporte».

#### Scenario: Text and photo
- **WHEN** a caller adds «Hoy en la mañana seguía igual» with a photo to a report
- **THEN** the aporte is created with that text and the photo
- **AND** the response carries its id, text, photo link, author name and date

#### Scenario: Text only
- **WHEN** a caller adds an aporte without a photo
- **THEN** the aporte is created without a photo

#### Scenario: Invalid data
- **WHEN** the text is empty or only spaces, the text exceeds 1200 characters, the author name exceeds 40 characters, or more than one photo is sent
- **THEN** the API answers 400 and creates nothing

#### Scenario: Unknown report
- **WHEN** the report does not exist, or the id is not valid
- **THEN** the API answers 404 and creates nothing

#### Scenario: No token
- **WHEN** the request carries no valid token
- **THEN** the API answers 401 and creates nothing

### Requirement: Adding an aporte also marks support
Adding an aporte is joining the report, so it SHALL also mark «También me afecta» for the caller on that report. The aporte and the support SHALL be saved together or not at all. A caller who already supports the report, or who is its author, SHALL get the aporte added and the count left as it was, without an error. The response SHALL carry the resulting affected count and the caller's support state.

#### Scenario: New supporter
- **GIVEN** a report that affects 14
- **WHEN** a neighbor who does not support it adds an aporte
- **THEN** the aporte is created, the report affects 15 and their state is «supporter»

#### Scenario: Already supporting
- **GIVEN** a neighbor who already supports a report that affects 15
- **WHEN** they add another aporte
- **THEN** the aporte is created and the report still affects 15

#### Scenario: Author adds an aporte
- **WHEN** the author of a report adds an aporte
- **THEN** the aporte is created and the affected count does not change

#### Scenario: Nothing half-saved
- **WHEN** the aporte cannot be saved
- **THEN** the support is not marked either

### Requirement: Aporte photos follow the report photo rules
The API SHALL accept only an aporte photo that can be read as a JPEG or PNG image within the same maximum size as report photos. It SHALL re-encode the photo so that the stored file carries no location nor other hidden file data, and SHALL serve it through a temporary link that expires after one hour. If the aporte cannot be saved, no stored photo SHALL be left behind.

#### Scenario: Photo with device location
- **WHEN** an aporte is added with a photo whose file carries the device location
- **THEN** the stored photo carries no location nor other hidden file data

#### Scenario: Not an image
- **WHEN** the photo cannot be read as an image
- **THEN** the API answers 400 and creates nothing

#### Scenario: Photo too large
- **WHEN** the photo exceeds the maximum size
- **THEN** the API answers 413 and creates nothing

#### Scenario: Database fails after storing the photo
- **WHEN** the aporte cannot be saved after its photo was stored
- **THEN** the API answers with a server error, no aporte exists and the stored photo is removed

### Requirement: Aportes are anonymous unless named
An aporte without an author name SHALL be public as «Vecino anónimo». The API SHALL record the user id of the session that added each aporte and SHALL NOT reveal it in any response.

#### Scenario: Anonymous aporte
- **WHEN** an aporte is added without an author name, or with only spaces
- **THEN** it has no public author name and no response carries the session's user id

### Requirement: Aporte rate is limited
The API SHALL allow one neighbor to add at most 10 aportes, across all reports, in any 10-minute window. Over that limit it SHALL answer 429 with how many seconds to wait, and create nothing.

#### Scenario: Limit exceeded
- **GIVEN** a neighbor who added 10 aportes in the last 10 minutes
- **WHEN** they add another one
- **THEN** the API answers 429 with how many seconds to wait and nothing is created

#### Scenario: Limit is per neighbor
- **GIVEN** one neighbor reached the limit
- **WHEN** a different neighbor adds an aporte
- **THEN** it is created

### Requirement: The API lists a report's aportes
The API SHALL return a report's aportes from oldest to newest in pages of at most the requested size (20 by default, between 1 and 50), with an opaque cursor for the next page, or none when there are no more. Each aporte SHALL carry its id, text, author name (empty when it has none), date and photo link (empty when it has no photo), and never a user id.

#### Scenario: Oldest first
- **GIVEN** aportes added at 10:00, 11:00 and 12:00
- **WHEN** a caller lists them
- **THEN** they come in the order 10:00, 11:00, 12:00

#### Scenario: Two pages
- **GIVEN** a report with 25 aportes
- **WHEN** a caller asks for the first page without a size and then for the page of its cursor
- **THEN** the first page has 20 aportes and a cursor, and the second has the other 5 and no cursor

#### Scenario: No aportes
- **WHEN** a report has no aportes
- **THEN** the API answers 200 with an empty list and no cursor

#### Scenario: Unknown report or invalid paging
- **WHEN** the report does not exist, the cursor is not one the API issued, or the size is out of range
- **THEN** the API answers 404 for the report, or 400 for the cursor or size

### Requirement: «Aportar» opens «Sumarte a este reporte»
The report page SHALL offer an «Aportar» button. It opens the same «Sumarte a este reporte» screen used from Reportar, with that report's card, the name switch, «Sumarme a este reporte» and «No es el mismo, volver». Here «Tu aporte» starts empty: the neighbor writes the text and can take or choose one photo. The card shows the distance only when the neighbor's location is already known. The photo SHALL be cleaned and reduced on the device before it is sent.

#### Scenario: Open the screen
- **GIVEN** a report that affects 14
- **WHEN** the neighbor taps «Aportar» on its page
- **THEN** «Sumarte a este reporte» shows the report's card with «a 14 les afecta», «Tu apoyo se suma a este caso; no se crea uno nuevo.», the switch «Mostrar mi nombre o apodo» off, and «Tu aporte» with an empty text field and a way to add one photo

#### Scenario: Distance unknown
- **GIVEN** the neighbor never shared their location
- **WHEN** they open the screen from «Aportar»
- **THEN** the card shows no distance

#### Scenario: Go back
- **GIVEN** the neighbor wrote part of an aporte
- **WHEN** they tap «No es el mismo, volver» or the back button
- **THEN** the report page is shown again and nothing is sent

#### Scenario: Empty text
- **WHEN** the neighbor taps «Sumarme a este reporte» with the text empty or only spaces
- **THEN** nothing is sent and a message in Spanish says the text is missing

#### Scenario: Switch on without a name
- **GIVEN** the switch is on and the name is blank
- **WHEN** the neighbor taps «Sumarme a este reporte»
- **THEN** nothing is sent and a message in Spanish asks for the name or to turn the switch off

#### Scenario: Replace the photo
- **GIVEN** the neighbor added a photo
- **WHEN** they remove it and add another one
- **THEN** only the new photo is kept

#### Scenario: Photo with device location
- **GIVEN** the neighbor chooses a photo whose file carries the device location
- **WHEN** it is added to «Tu aporte»
- **THEN** the photo kept for sending carries no location nor other hidden file data

### Requirement: Publishing an aporte adds it to the list
When the aporte is created, the app SHALL open the report page with the notice «Te sumaste al reporte», the aporte in the list, the counts updated and «También me afecta» marked. While sending, «Sumarme a este reporte» SHALL be disabled. On failure the app SHALL keep the screen as it was and show a message in Spanish, which asks the neighbor to wait when there are too many aportes.

#### Scenario: Aporte published
- **GIVEN** a report that affects 14 and the neighbor does not support it
- **WHEN** the neighbor sends «Sigue igual hoy» with a photo from «Aportar»
- **THEN** the report page opens with «Te sumaste al reporte» and «Ahora a 15 vecinos les afecta. Tu aporte ya aparece abajo.»
- **AND** «Sigue igual hoy» appears in the list with its photo, «Vecino anónimo» and «hace un momento», and «También me afecta · 15» is marked

#### Scenario: Publish fails
- **GIVEN** the API cannot be reached
- **WHEN** the neighbor taps «Sumarme a este reporte»
- **THEN** a message in Spanish says they could not join and the screen keeps the text and photo

#### Scenario: Too many aportes
- **GIVEN** the API answers that the neighbor added too many aportes
- **WHEN** they tap «Sumarme a este reporte»
- **THEN** a message in Spanish asks them to wait before adding another

### Requirement: The report page lists the aportes
The report page SHALL show the section «Aportes y comentarios · N» with the report's aportes from oldest to newest. Each aporte SHALL show its author (or «Vecino anónimo»), its date in the same format as the report date, its text and, when it has one, its photo. More aportes SHALL load when the neighbor asks for them. A photo that cannot be loaded SHALL be replaced by a placeholder.

#### Scenario: List
- **GIVEN** a report with an aporte by «Rosa» from 2 hours ago with a photo, and an anonymous one without a photo
- **WHEN** a neighbor opens the report
- **THEN** «Aportes y comentarios · 2» lists «Rosa · hace 2 h» with its text and photo, then «Vecino anónimo» with its text

#### Scenario: No aportes
- **WHEN** a report has no aportes
- **THEN** the section shows «Aportes y comentarios · 0» and a message in Spanish inviting to add one

#### Scenario: More aportes
- **GIVEN** a report with 25 aportes
- **WHEN** the neighbor asks for more at the end of the list
- **THEN** the other 5 appear below

#### Scenario: List fails
- **WHEN** the aportes cannot be loaded
- **THEN** the section shows a message in Spanish with «Reintentar» and the rest of the page stays as it is
