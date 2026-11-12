# Spec Delta

## MODIFIED Requirements

### Requirement: Search results carry what a card shows
Each result SHALL carry the same data as an item of the feed: id, title, category, status, place reference, publication date, a link to its first photo that expires after one hour, its affected count, and its distance when the caller sent a location. Results SHALL NOT carry the details, the author name, or any user id.

#### Scenario: Affected count
- **GIVEN** a matching report that affects 6 neighbors
- **WHEN** a caller searches
- **THEN** its result carries the affected count 6, and its card in Buscar shows «6» next to a people icon

#### Scenario: The publisher is never revealed
- **WHEN** any caller searches reports
- **THEN** no result carries an author name or the user id of the session that published it
