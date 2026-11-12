# Spec Delta

## MODIFIED Requirements

### Requirement: The API returns one report
The API SHALL return, for a caller with a valid session, a report by its id with its title, details, category, status, location, place reference, publication date, author name (empty when the report has none), its photos in the order they were published, its affected count, its number of aportes and the caller's support state (author, supporter or none), and SHALL answer 404 when the report does not exist.

#### Scenario: Existing report
- **GIVEN** a published report
- **WHEN** a caller with a valid session asks for it by id
- **THEN** the API returns its title, details, category, status «publicado», location, reference, date, author name and photos

#### Scenario: Affected count, aportes and support state
- **GIVEN** a report that 3 neighbors besides its author support and that has 2 aportes
- **WHEN** a caller who supports it asks for it
- **THEN** the API returns the affected count 4, the aporte count 2 and the support state «supporter»
- **WHEN** the session that published it asks for it
- **THEN** the support state is «author»
- **WHEN** any other session asks for it
- **THEN** the support state is «none»

#### Scenario: Unknown report
- **WHEN** the id does not match any report, or is not a valid id
- **THEN** the API answers 404

#### Scenario: No token
- **WHEN** the request carries no valid token
- **THEN** the API answers 401

#### Scenario: The publisher is never revealed
- **WHEN** any caller asks for a report published without an author name
- **THEN** the response carries no author name and no user id, whichever session published it

### Requirement: The report page shows the report
The report page SHALL show the report's photos, category, status, title, author, publication date, place reference and details, then the «También me afecta» and «Aportar» buttons, the place on a map and the aportes list, for any neighbor with or without an account. Parts the report does not have (details, reference) SHALL NOT be shown empty.

#### Scenario: Open a report
- **GIVEN** a report in Limpieza, «Publicado», published 2 hours ago by «Rosa», with the reference «Jr. Ejemplo cdra. 3», details, 2 photos, 14 affected and 5 aportes
- **WHEN** a neighbor opens its page
- **THEN** they see its photos, «Limpieza», «Publicado», the title, «Rosa · hace 2 h · Jr. Ejemplo cdra. 3» and the details
- **AND** below them «También me afecta · 14» and «Aportar»
- **AND** a map with a pin at the report's location and «Aportes y comentarios · 5»

#### Scenario: Without optional parts
- **GIVEN** a report without details and without a reference
- **WHEN** a neighbor opens its page
- **THEN** no empty details nor empty reference are shown

#### Scenario: Status labels
- **WHEN** the page shows the status
- **THEN** it uses the same Spanish labels as the cards of Inicio

## ADDED Requirements

### Requirement: The report page confirms joining
When the neighbor arrives at a report page right after joining it, the page SHALL show the success notice «Te sumaste al reporte» with «Ahora a N vecinos les afecta. Tu aporte ya aparece abajo.», where N is the new affected count, like «Reporte publicado» after publishing. It SHALL NOT show it again when the page is reloaded or opened later.

#### Scenario: After joining
- **GIVEN** a report that affected 14 neighbors
- **WHEN** the neighbor joins it from Reportar
- **THEN** its page opens with the notice «Te sumaste al reporte» and «Ahora a 15 vecinos les afecta. Tu aporte ya aparece abajo.»

#### Scenario: Reload
- **GIVEN** the neighbor saw «Te sumaste al reporte»
- **WHEN** they reload the page
- **THEN** the notice is not shown again
