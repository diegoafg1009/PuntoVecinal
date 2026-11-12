# Spec Delta

## RENAMED Requirements

- FROM: `### Requirement: Step 1 shows the emergency notice`
- TO: `### Requirement: Every step shows the emergency notice`

## MODIFIED Requirements

### Requirement: Every step shows the emergency notice
Every step of the Reportar form SHALL show, at the top and without a way to close it, a notice that the app does not handle emergencies. It SHALL tell the neighbor to call Serenazgo or the official emergency numbers 105 (Policía), 116 (Bomberos) and 106 (SAMU). Each number SHALL be a link that opens a call to it.

#### Scenario: Notice on step 1
- **WHEN** the neighbor is on step 1
- **THEN** the notice is shown at the top of the step with the title «¿Es una emergencia o un tema de seguridad?»
- **AND** it says that the app does not handle emergencies and names Serenazgo, 105 Policía, 116 Bomberos and 106 SAMU

#### Scenario: Notice on the other steps
- **WHEN** the neighbor moves to step 2 or step 3
- **THEN** the same notice is shown at the top of that step

#### Scenario: The notice cannot be closed
- **WHEN** the neighbor is on any step of Reportar
- **THEN** the notice has no button to close it

#### Scenario: Call a number
- **GIVEN** the neighbor is on a phone
- **WHEN** they tap «105» in the notice
- **THEN** the phone offers to call 105

## ADDED Requirements

### Requirement: Reports cannot be about safety
The categories offered in Reportar SHALL NOT include safety or emergencies, so there is no way to publish a safety report.

#### Scenario: Categories offered
- **WHEN** the neighbor looks at the categories in step 1
- **THEN** they are Limpieza, Alumbrado, Pistas y veredas, Agua y desagüe and Otro, and none of them is about safety
