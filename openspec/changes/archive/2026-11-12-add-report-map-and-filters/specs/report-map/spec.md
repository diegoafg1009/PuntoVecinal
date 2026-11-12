# Spec Delta

## Purpose

Defines how neighbors, with or without an account, see the published reports as points on a map (HU-06, PV-15). It covers the API operation that returns the points and the Mapa screen that shows them, colors them by status, previews a report and centers on the neighbor or on the pilot zone.

## ADDED Requirements

### Requirement: The API returns the reports as map points
The API SHALL return to a caller with a valid session the published reports as map points, newest first and without pages, with at most the 500 newest reports. Each point SHALL carry the report id, title, category (slug and label), status and the location of the problem. It SHALL NOT carry the details, the author name, any user id or any location other than the report's own.

#### Scenario: Points of the reports
- **GIVEN** two published reports
- **WHEN** a caller with a valid session asks for the map points
- **THEN** the API answers 200 with two points
- **AND** each carries its id, title, category, status, latitude and longitude

#### Scenario: Anonymous session
- **GIVEN** a caller with an anonymous session
- **WHEN** they ask for the map points
- **THEN** the API answers 200 with the points

#### Scenario: More than 500 reports
- **GIVEN** 510 published reports
- **WHEN** a caller asks for the map points
- **THEN** the API returns the 500 newest

#### Scenario: No reports yet
- **WHEN** no report has been published and a caller asks for the map points
- **THEN** the API answers 200 with no points

#### Scenario: The publisher is never revealed
- **WHEN** any caller asks for the map points
- **THEN** no point carries an author name nor the user id of the session that published it

#### Scenario: No token
- **WHEN** the request carries no valid token
- **THEN** the API answers 401

### Requirement: Mapa shows one point per report
Mapa SHALL show a map with one pin per report that the API returned, placed at the report's location, which is where the problem is and never where its author was. Each pin's color SHALL depend on its status, and the screen SHALL show a legend that names each color with the Spanish status label.

#### Scenario: Open Mapa
- **WHEN** a neighbor, with or without an account, opens Mapa
- **THEN** they see a map with one point per report

#### Scenario: Color by status
- **GIVEN** a report «Publicado» and another «En atención»
- **WHEN** Mapa shows them
- **THEN** their points have different colors
- **AND** the legend says which color is «Publicado» and which is «En atención»

#### Scenario: Point at the place of the problem
- **GIVEN** a report published from one place, with the pin moved to another place
- **WHEN** Mapa shows it
- **THEN** its point is at the place where the pin was left

### Requirement: Tapping a pin previews the report
Tapping a pin SHALL open a preview card at the bottom of the map with the report's first photo, title, category and status. The whole card SHALL lead to the page of that report. The preview SHALL show a placeholder if the photo cannot be shown, and SHALL NOT show a distance. Tapping the map outside a pin SHALL close the preview.

#### Scenario: Preview
- **WHEN** the neighbor taps the pin of «Poste sin luz en Jr. Ejemplo»
- **THEN** a preview shows «Poste sin luz en Jr. Ejemplo», its category, its status and its first photo

#### Scenario: Go to the report
- **GIVEN** the preview of a report is open
- **WHEN** the neighbor taps the card
- **THEN** the page of that report opens

#### Scenario: Close the preview
- **GIVEN** the preview of a report is open
- **WHEN** the neighbor taps the map outside a pin
- **THEN** the preview closes

#### Scenario: Photo cannot be shown
- **WHEN** the photo of the preview cannot be loaded
- **THEN** the preview shows a placeholder in its place and keeps the title, category and status

### Requirement: Mapa centers on the neighbor or on the pilot zone
Mapa SHALL center on the neighbor's location once they share it. It SHALL ask for the device location only when the neighbor taps the locate button, which SHALL be labeled «Usar mi ubicación» for assistive technology. It SHALL reuse a location shared before, without asking, while the browser keeps the permission. Without a location, Mapa SHALL center on the pilot zone.

#### Scenario: No location
- **WHEN** a neighbor who never shared their location opens Mapa
- **THEN** the app does not ask for the device location
- **AND** the map is centered on the pilot zone

#### Scenario: Share the location
- **WHEN** the neighbor taps the locate button on Mapa and accepts
- **THEN** the map centers on where they are

#### Scenario: Previously shared location
- **GIVEN** the neighbor shared their location before and the browser still allows it
- **WHEN** they open Mapa
- **THEN** the map centers on where they are without asking again

#### Scenario: Decline or failure
- **WHEN** the neighbor taps the locate button and declines, or the position cannot be obtained
- **THEN** a message in Spanish says the location could not be obtained
- **AND** the map stays centered on the pilot zone

### Requirement: Mapa shows loading and error states
Mapa SHALL show the map with a loading indicator while the points arrive. When the points cannot be loaded, it SHALL show a message in Spanish with «Reintentar». When there are no reports at all, it SHALL say there are no reports yet and offer «Reportar».

#### Scenario: Loading
- **WHEN** Mapa opens and the points have not arrived yet
- **THEN** the map is shown with a loading indicator

#### Scenario: Load failure and retry
- **GIVEN** the API cannot be reached when Mapa opens
- **THEN** a message in Spanish says the reports could not be loaded and offers «Reintentar»
- **WHEN** the neighbor taps «Reintentar» and the API answers
- **THEN** the points are shown

#### Scenario: No reports
- **WHEN** no report has been published and no filter is on
- **THEN** Mapa says there are no reports yet and offers «Reportar»
