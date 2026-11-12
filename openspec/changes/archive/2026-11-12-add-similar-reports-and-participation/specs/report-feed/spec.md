# Spec Delta

## MODIFIED Requirements

### Requirement: Each listed report carries what a card shows
Each item of the list SHALL carry the report id, title, category (slug and label), status, place reference (null when there is none), publication date, the link to its first photo, which expires after one hour, and its affected count as `report-support` defines it. Items SHALL NOT carry the details, the author name nor any user id.

#### Scenario: Item contents
- **GIVEN** a report with 2 photos and the reference «frente al mercado»
- **WHEN** it is listed
- **THEN** its item carries its id, title, category, status, the reference, its publication date and one link that shows its first photo

#### Scenario: Affected count
- **GIVEN** a report that 13 neighbors besides its author support
- **WHEN** it is listed
- **THEN** its item carries the affected count 14

#### Scenario: The publisher is never revealed
- **WHEN** any caller lists the reports
- **THEN** no item carries an author name nor the user id of the session that published it

### Requirement: Inicio shows the reports as cards
Inicio SHALL show the published reports from newest to oldest as cards. Each card SHALL show the first photo, the title, the category, the status, the place reference when there is one, the distance when known, how long ago the report was published, and the affected count next to a people icon.

#### Scenario: Open Inicio
- **WHEN** a neighbor, with or without an account, opens Inicio
- **THEN** they see the reports as cards, the most recent first

#### Scenario: Card contents
- **GIVEN** a report «Basura acumulada en la esquina del parque» in Limpieza, «Publicado», published 2 hours ago, with the reference «Jr. Ejemplo cdra. 3», that affects 14 neighbors
- **WHEN** its card is shown without the neighbor's location
- **THEN** it shows its photo, the title, «Limpieza», «Publicado», «Jr. Ejemplo cdra. 3», «hace 2 h» and «14» next to a people icon

#### Scenario: New report
- **WHEN** the card of a report nobody else supports is shown
- **THEN** it shows «1» next to the people icon

#### Scenario: Status labels
- **WHEN** a card shows a status
- **THEN** it uses the Spanish label of the status: «Publicado», «Derivado», «Recibido por la entidad», «En atención», «Atendido según la entidad» or «Cerrado»

#### Scenario: Photo cannot be shown
- **WHEN** the photo of a card cannot be loaded
- **THEN** the card shows a placeholder in its place and the rest of the card stays as it is
