# Spec Delta

## Purpose

Defines how neighbors narrow the published reports by category and status in Inicio and Mapa (HU-07, PV-16). It covers the filters the feed and map operations accept, the filter chips, the filters both sections share and the message shown when nothing matches.

## ADDED Requirements

### Requirement: The API filters reports by category
The feed and map-point operations SHALL accept an optional list of one or more category slugs and return only the reports in any of them. With no category, they return reports of every category. An unknown slug SHALL be rejected with 400.

#### Scenario: One category
- **GIVEN** a report in Limpieza and another in Alumbrado
- **WHEN** a caller lists the reports filtering by «limpieza»
- **THEN** only the Limpieza report is returned

#### Scenario: Several categories
- **GIVEN** reports in Limpieza, Alumbrado and Agua y desagüe
- **WHEN** a caller asks for the map points filtering by «limpieza» and «alumbrado»
- **THEN** only the Limpieza and Alumbrado points are returned

#### Scenario: Unknown category
- **WHEN** a caller filters by a category slug that is not one of the fixed categories
- **THEN** the API answers 400

### Requirement: The API filters reports by status
The feed and map-point operations SHALL accept one optional status and return only the reports in that status. With no status, they return reports in every status. A value that is not a report status SHALL be rejected with 400.

#### Scenario: One status
- **GIVEN** a report «Publicado» and another «En atención»
- **WHEN** a caller lists the reports filtering by «en_atencion»
- **THEN** only the «En atención» report is returned

#### Scenario: Category and status together
- **GIVEN** a Limpieza report «Publicado», a Limpieza report «Atendido según la entidad» and an Alumbrado report «Publicado»
- **WHEN** a caller filters by «limpieza» and «publicado»
- **THEN** only the Limpieza report «Publicado» is returned

#### Scenario: Unknown status
- **WHEN** a caller filters by a status that is not a report status
- **THEN** the API answers 400

### Requirement: Filtered feed pages keep their order and cursor
A filtered feed SHALL keep the feed's order and pagination. Each page holds only matching reports, and the cursor of a page SHALL continue with the next matching report, without repeating or skipping any.

#### Scenario: Two filtered pages
- **GIVEN** 25 Limpieza reports and 10 Alumbrado reports
- **WHEN** a caller walks through the feed pages filtering by «limpieza», passing the same filter with each cursor
- **THEN** the pages hold the 25 Limpieza reports, newest first, each exactly once

### Requirement: Inicio and Mapa offer filter chips
Inicio and Mapa SHALL each show a row of filter chips: «Todo», «Estado» and one chip per fixed category. A chip SHALL take effect as soon as it is tapped or chosen, with no confirmation step. Active chips SHALL look different from inactive ones, and the row SHALL scroll horizontally when it does not fit.

#### Scenario: Chips shown
- **WHEN** Inicio or Mapa opens
- **THEN** the chip row shows «Todo», «Estado» and one chip per category

#### Scenario: Filter by categories
- **GIVEN** the neighbor is on Inicio
- **WHEN** they tap the Limpieza chip and then the Alumbrado chip
- **THEN** Inicio shows only Limpieza and Alumbrado reports
- **AND** both chips look active and «Todo» does not

#### Scenario: Turn a category off
- **GIVEN** the Limpieza chip is active
- **WHEN** the neighbor taps it again
- **THEN** it is no longer active and its category is not filtered

### Requirement: Chips choose categories and status
Tapping a category chip SHALL turn that category on or off, so one or more can be on. The «Estado» chip SHALL open a menu with «Todos» and the statuses, of which one can be chosen, and SHALL name the chosen status. «Todo» SHALL look active when no filter is on, and tapping it SHALL clear every filter.

#### Scenario: Filter by status
- **GIVEN** the neighbor is on Mapa
- **WHEN** they open «Estado» and choose «En atención»
- **THEN** Mapa shows only the pins of reports «En atención»
- **AND** the «Estado» chip looks active and names «En atención»

#### Scenario: Status labels
- **WHEN** the «Estado» menu is open
- **THEN** it lists «Todos», «Publicado», «Derivado», «Recibido por la entidad», «En atención», «Atendido según la entidad» and «Cerrado»

#### Scenario: Todo clears the filters
- **GIVEN** the neighbor has a category and a status on
- **WHEN** they tap «Todo»
- **THEN** every report is shown again
- **AND** «Todo» looks active and no other chip does

#### Scenario: Category chips follow the API
- **WHEN** the categories arrive
- **THEN** there is one chip per category, with its Spanish label, in the order the API gives them

### Requirement: The filters are shared by Inicio and Mapa
The filters in effect SHALL be the same in Inicio and Mapa, and SHALL be kept when moving between sections or opening a report during the visit. A reload SHALL clear them.

#### Scenario: From Inicio to Mapa
- **GIVEN** the neighbor applied the Limpieza filter on Inicio
- **WHEN** they open Mapa
- **THEN** Mapa shows only Limpieza points
- **AND** its Limpieza chip looks active

#### Scenario: Back from a report
- **GIVEN** the neighbor applied a filter on Inicio and opened a report
- **WHEN** they go back to Inicio
- **THEN** the filter is still in effect

#### Scenario: Reload
- **GIVEN** filters are in effect
- **WHEN** the neighbor reloads the page
- **THEN** no filter is in effect

### Requirement: Filters can be cleared
The filters SHALL be cleared from the «Todo» chip, which clears every filter in effect in Inicio and Mapa, and from the empty state described below.

#### Scenario: Clear with Todo
- **GIVEN** filters are in effect
- **WHEN** the neighbor taps «Todo»
- **THEN** every report is shown again
- **AND** no category or status chip looks active

### Requirement: Nothing matches the filters
When filters are in effect and no report matches them, Inicio and Mapa SHALL say in Spanish that no report matches the filters and SHALL offer «Quitar filtros».

#### Scenario: No match on Inicio
- **GIVEN** there are reports, but none in Alumbrado
- **WHEN** the neighbor applies the Alumbrado filter on Inicio
- **THEN** Inicio says no report matches the filters and offers «Quitar filtros»

#### Scenario: No match on Mapa
- **GIVEN** there are reports, but none «Cerrado»
- **WHEN** the neighbor applies the «Cerrado» filter on Mapa
- **THEN** the map shows no points, a message says no report matches the filters, and «Quitar filtros» is offered

#### Scenario: Clear from the empty state
- **GIVEN** nothing matches the filters
- **WHEN** the neighbor taps «Quitar filtros»
- **THEN** every report is shown again
