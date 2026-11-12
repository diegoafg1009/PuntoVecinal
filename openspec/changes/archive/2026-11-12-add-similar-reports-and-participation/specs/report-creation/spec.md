# Spec Delta

## MODIFIED Requirements

### Requirement: Step 2 validates photos and place
The app SHALL NOT leave step 2 forward while there is no photo or no location has been marked, and SHALL tell the neighbor in Spanish what is missing. With photos and a location, «Seguir» SHALL first look for similar reports, as `similar-reports` describes, showing that it is working and ignoring further taps meanwhile. It SHALL move to step 3 only when there are none, or when the check fails. The neighbor SHALL be able to go back to step 1 at any time without losing what they entered.

#### Scenario: No photo
- **GIVEN** the neighbor placed the pin but added no photo
- **WHEN** they tap «Seguir»
- **THEN** a message in Spanish says that at least one photo is needed
- **AND** the form stays on step 2

#### Scenario: No location
- **GIVEN** the neighbor added a photo but did not move the pin
- **WHEN** they tap «Seguir»
- **THEN** a message in Spanish says that the place is missing
- **AND** the form stays on step 2

#### Scenario: Complete step 2
- **GIVEN** the neighbor has 1 to 3 photos and a location, and there is no similar report nearby
- **WHEN** they tap «Seguir»
- **THEN** the form moves to step 3, «Revisar», without showing any dialog

#### Scenario: Complete step 2 with similar reports
- **GIVEN** the neighbor has 1 to 3 photos and a location, and there is a similar report nearby
- **WHEN** they tap «Seguir»
- **THEN** the dialog «¿Ya lo reportó alguien?» opens over step 2

#### Scenario: Checking
- **WHEN** the neighbor taps «Seguir» and the check has not answered yet
- **THEN** «Seguir» shows that it is working and another tap does not start a second check

#### Scenario: Go back
- **GIVEN** the neighbor is on step 2 with photos and a location
- **WHEN** they go back to step 1 and tap «Seguir» again
- **THEN** step 2 shows the same photos, location and reference
