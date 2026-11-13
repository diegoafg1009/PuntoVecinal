# Spec Delta

## Purpose

Defines how a neighbor avoids reporting a problem twice (HU-13, PV-22 and HU-14, PV-23). Before publishing, they see open reports near the place they marked that mean the same thing, and they can join one of them instead of publishing a new report.

## ADDED Requirements

### Requirement: The API finds similar reports near a place
The API SHALL return, for a caller with a valid session who sends a title, optional details and a location, the reports within 300 m of that location whose text is close in meaning, best match first, at most 5. A report whose text is not close enough SHALL NOT be returned, however near it is.

#### Scenario: Same problem in other words
- **GIVEN** a report titled «No hay alumbrado en el pasaje» 80 m away
- **WHEN** a caller checks «Poste sin luz frente a mi casa» at their location
- **THEN** that report is among the first results

#### Scenario: Unrelated report nearby
- **GIVEN** a report titled «Fuga de agua en la esquina» 50 m away, and no other report nearby
- **WHEN** a caller checks «Basura acumulada en el parque»
- **THEN** the API answers 200 with an empty list

#### Scenario: Similar report too far
- **GIVEN** a report with the same text 450 m away
- **WHEN** a caller checks that text
- **THEN** that report is not returned

#### Scenario: At most five
- **GIVEN** 7 similar reports within 300 m
- **WHEN** a caller checks
- **THEN** 5 reports are returned, best match first

#### Scenario: Invalid input
- **WHEN** the title is blank or longer than 100 characters, the details exceed 1000 characters, or the location is missing or out of range
- **THEN** the API answers 400

#### Scenario: No token
- **WHEN** the request carries no valid token
- **THEN** the API answers 401

### Requirement: Only open reports are offered as similar
The API SHALL NOT return as similar a report whose status is «Atendido según la entidad» or «Cerrado».

#### Scenario: Closed report
- **GIVEN** a similar report 50 m away whose status is «Cerrado»
- **WHEN** a caller checks
- **THEN** that report is not returned

#### Scenario: Forwarded report
- **GIVEN** a similar report 50 m away whose status is «Derivado»
- **WHEN** a caller checks
- **THEN** that report is returned

### Requirement: Similar results carry what the dialog shows
Each similar report SHALL carry its id, title, category, status, publication date, a link to its first photo that expires after one hour, its distance in meters from the sent location, and its affected count. Results SHALL NOT carry the details, the author name or any user id.

#### Scenario: Result content
- **GIVEN** a similar report in Limpieza, 50 m away, published 2 hours ago, that affects 14 neighbors
- **WHEN** a caller checks
- **THEN** its result carries its title, «Limpieza», its status, its date, a photo link, 50 and 14

#### Scenario: The publisher is never revealed
- **WHEN** any caller checks similar reports
- **THEN** no result carries an author name or the user id of the session that published it

### Requirement: Only the report text is compared
To compare reports, the system SHALL send to the text-comparison service only the title and the details of a report or of a draft, never the author name, the place reference, the location, photos or any user data.

#### Scenario: Data sent for a check
- **WHEN** a caller checks a draft with an author name, a reference and photos
- **THEN** only its title and details leave the system for comparison

#### Scenario: Data sent for a published report
- **WHEN** a report is published with the author name «Rosa»
- **THEN** only its title and details leave the system for comparison

### Requirement: Published reports become comparable
Every published report SHALL become comparable as a similar report. Publishing SHALL NOT fail or wait more than a few seconds because the text-comparison service is unavailable. A report published while that service is unavailable SHALL become comparable once the service is back.

#### Scenario: Comparable right after publishing
- **GIVEN** the text-comparison service is available
- **WHEN** a neighbor publishes «Poste sin luz en el pasaje»
- **THEN** a check of «No hay alumbrado en el pasaje» at that place returns it

#### Scenario: Service down while publishing
- **GIVEN** the text-comparison service cannot be reached
- **WHEN** a neighbor publishes a report
- **THEN** the report is published as usual
- **AND** once the service is reachable again, the report is returned by a matching check

### Requirement: A failed check does not block publishing
When the similar reports cannot be checked, the API SHALL answer with a server error and create nothing. The app SHALL then move the neighbor on to «Revisar» as if there were no similar reports, without an error message.

#### Scenario: Service down during a check
- **GIVEN** the text-comparison service cannot be reached
- **WHEN** the neighbor taps «Seguir» on step 2 with valid photos and place
- **THEN** step 3, «Revisar», opens with everything they entered

### Requirement: «¿Ya lo reportó alguien?» shows the similar reports
When the check returns similar reports, the app SHALL open the dialog «¿Ya lo reportó alguien?». It explains that there are similar reports within 300 m of the marked place and offers to join one instead of creating a new report. It lists each report with its photo, title, category, distance, affected count and how long ago it was published, in a list that scrolls.

#### Scenario: Dialog content
- **GIVEN** a similar report «Basura acumulada en la esquina del parque» in Limpieza, 50 m away, affecting 14, published 2 hours ago
- **WHEN** the dialog opens
- **THEN** that report shows its photo, its title, «Limpieza», «50 m», «14» and «hace 2 h»

#### Scenario: Many results
- **GIVEN** 5 similar reports that do not fit on the screen
- **WHEN** the neighbor scrolls the list
- **THEN** every result can be reached and the dialog's buttons stay reachable

#### Scenario: Photo cannot be loaded
- **WHEN** the photo of a result cannot be loaded
- **THEN** a placeholder is shown in its place

#### Scenario: Mobile and desktop
- **WHEN** the dialog opens
- **THEN** it is a bottom sheet on mobile and a centered dialog on desktop

### Requirement: «No es ninguno, seguir» continues to «Revisar»
The dialog SHALL offer «No es ninguno, seguir», which closes it and opens step 3, «Revisar», with everything the neighbor entered. Closing the dialog without choosing SHALL keep the neighbor on step 2 with everything they entered.

#### Scenario: None of them
- **GIVEN** the dialog is open
- **WHEN** the neighbor taps «No es ninguno, seguir»
- **THEN** step 3 opens with their category, title, details, photos and place

#### Scenario: Dismiss the dialog
- **GIVEN** the dialog is open
- **WHEN** the neighbor closes it without choosing
- **THEN** the form stays on step 2 with everything they entered

### Requirement: Joining shows the report and «Tu aporte»
Tapping a report in the dialog SHALL open «Sumarte a este reporte». The screen SHALL show a compact card of that report with its category, status, title, distance and «a N les afecta», plus the text «Tu apoyo se suma a este caso; no se crea uno nuevo.». It SHALL also show «Tu aporte», which holds the draft's «¿Qué pasa?», its «Más detalles» when there are any, and its first photo.

#### Scenario: Open the join screen
- **GIVEN** the draft «Hay bolsas de basura en la esquina» with 2 photos
- **WHEN** the neighbor taps «Basura acumulada en la esquina del parque» in the dialog
- **THEN** «Sumarte a este reporte» shows that report's card with «a 14 les afecta»
- **AND** «Tu aporte» shows «Hay bolsas de basura en la esquina» and the first of the 2 photos

#### Scenario: Not the same
- **GIVEN** the join screen is open
- **WHEN** the neighbor taps «No es el mismo, volver» or the back button
- **THEN** the dialog «¿Ya lo reportó alguien?» is shown again over step 2, with the same results

### Requirement: Joining is anonymous unless the neighbor shows a name
For a neighbor without an account, the join screen SHALL offer the switch «Mostrar mi nombre o apodo», off by default. With it off, the aporte is shown as «Vecino anónimo». With it on, a name field of up to 40 characters appears, and the aporte is shown with that name. With the switch on and the name blank, the app SHALL NOT join and SHALL ask for the name in Spanish.

#### Scenario: Default is anonymous
- **WHEN** the neighbor joins without touching the switch
- **THEN** their aporte is shown as «Vecino anónimo»

#### Scenario: Show a name
- **WHEN** the neighbor turns the switch on, writes «Rosa» and joins
- **THEN** their aporte is shown as «Rosa»

#### Scenario: Switch on without a name
- **GIVEN** the switch is on and the name is empty or only spaces
- **WHEN** the neighbor taps «Sumarme a este reporte»
- **THEN** nothing is sent and a message in Spanish asks for the name or to turn the switch off

### Requirement: Joining adds support and the aporte without a new report
Confirming with «Sumarme a este reporte» SHALL, in one request, mark «También me afecta» for the neighbor on that report and add «Tu aporte» to it, and SHALL NOT create a report. Both happen or neither does. The app SHALL then clear the draft and open the report page with the notice «Te sumaste al reporte» and the support marked, as `report-detail` describes. While the request is in progress the button SHALL be disabled.

#### Scenario: Join
- **GIVEN** a report that affects 14 neighbors and has 5 aportes
- **WHEN** the neighbor confirms joining it
- **THEN** its page opens with the notice «Te sumaste al reporte» and «Ahora a 15 vecinos les afecta. Tu aporte ya aparece abajo.»
- **AND** «También me afecta · 15» is marked and their aporte is in the list
- **AND** no new report was created
- **AND** opening Reportar again shows an empty form on step 1

#### Scenario: Already supporting
- **GIVEN** the neighbor had already marked «También me afecta» on that report
- **WHEN** they join it
- **THEN** their aporte is added and the affected count does not change

#### Scenario: Join fails
- **GIVEN** the API cannot be reached or answers with an error
- **WHEN** the neighbor confirms joining
- **THEN** a message in Spanish says they could not join and offers to try again
- **AND** the draft and the join screen stay as they were

#### Scenario: Double tap
- **WHEN** the neighbor taps «Sumarme a este reporte» twice quickly
- **THEN** only one request is sent
