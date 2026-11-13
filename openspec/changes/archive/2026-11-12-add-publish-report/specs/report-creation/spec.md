# Spec Delta

## ADDED Requirements

### Requirement: Step 3 reviews the report
Step 3, «Revisar», SHALL show a summary with the category, the title, the details, the photos and the place, SHALL let the neighbor go back to any step to edit it without losing what they entered, and SHALL offer a «Publicar» button.

#### Scenario: Summary
- **GIVEN** the neighbor completed steps 1 and 2
- **WHEN** they reach step 3
- **THEN** they see the category, the title, the details (if any), the photos and the place with its reference (if any)

#### Scenario: Edit from the summary
- **GIVEN** the neighbor is on step 3
- **WHEN** they choose to edit the title and go to step 1
- **THEN** step 1 shows what they had written
- **AND** after changing it and continuing through step 2 they see the new title on step 3

#### Scenario: Optional parts missing
- **GIVEN** the neighbor left «Más detalles» and the place reference empty
- **WHEN** they reach step 3
- **THEN** the summary does not show empty details nor an empty reference

### Requirement: Publishing requires accepting the community rules
The app SHALL NOT publish the report until the neighbor accepts the community rules with a checkbox, and when they tap «Publicar» without accepting it SHALL point at the checkbox with a message in Spanish.

#### Scenario: Rules not accepted
- **GIVEN** the neighbor is on step 3 and did not accept the rules
- **WHEN** they tap «Publicar»
- **THEN** nothing is sent
- **AND** the checkbox is marked with a message in Spanish

#### Scenario: Rules accepted
- **GIVEN** the neighbor accepted the rules
- **WHEN** they tap «Publicar»
- **THEN** the report is sent

#### Scenario: Read the rules
- **WHEN** the neighbor opens the community rules from step 3
- **THEN** the rules are shown and the neighbor returns to step 3 when closing them

### Requirement: Neighbors without an account choose how they appear
For a neighbor without an account, step 3 SHALL offer the switch «Mostrar mi nombre o apodo», off by default. With it off, the report is published as «Vecino anónimo». With it on, a name field (up to 40 characters) appears and the report is published with that name.

#### Scenario: Default is anonymous
- **GIVEN** a neighbor without an account on step 3
- **WHEN** they publish without touching the switch
- **THEN** the report is sent without an author name
- **AND** step 3 says the report will appear as «Vecino anónimo»

#### Scenario: Show a name
- **GIVEN** a neighbor without an account
- **WHEN** they turn the switch on, write «Rosa» and publish
- **THEN** the report is sent with the author name «Rosa»

#### Scenario: Switch on without a name
- **GIVEN** the switch is on and the name field is empty or only spaces
- **WHEN** the neighbor taps «Publicar»
- **THEN** nothing is sent
- **AND** a message in Spanish asks for the name or to turn the switch off

#### Scenario: Switch turned off again
- **GIVEN** the neighbor turned the switch on and wrote a name
- **WHEN** they turn it off and publish
- **THEN** the report is sent without an author name

### Requirement: Publishing outcome
When the report is published, the app SHALL clear the draft, open the page of the new report and show the notice «Reporte publicado». While the request is in progress the app SHALL disable «Publicar» so the report is not sent twice.

#### Scenario: Successful publish
- **GIVEN** the neighbor completed the three steps and accepted the rules
- **WHEN** they tap «Publicar» and the API creates the report
- **THEN** the page of the new report opens with the notice «Reporte publicado»
- **AND** opening Reportar again shows an empty form on step 1 without photos or location

#### Scenario: Double tap
- **WHEN** the neighbor taps «Publicar» twice quickly
- **THEN** only one report is sent

#### Scenario: Publish fails
- **GIVEN** the API cannot be reached or answers with an error
- **WHEN** the neighbor taps «Publicar»
- **THEN** a message in Spanish says the report could not be published and offers to try again
- **AND** the form stays on step 3 with everything they entered

#### Scenario: Too many reports
- **GIVEN** the API answers that the neighbor published too many reports in a short time
- **WHEN** they tap «Publicar»
- **THEN** a message in Spanish asks them to wait before publishing again
- **AND** the form stays on step 3 with everything they entered
