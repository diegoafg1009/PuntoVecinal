# report-search Specification

## Purpose
Defines how neighbors, with or without an account, find published reports by words and by closeness (HU-04, PV-13): the paginated API operation that searches them and the Buscar screen that shows the results.

## Requirements

### Requirement: The API searches reports by words
The API SHALL return, for a caller with a valid session, the published reports whose title or details contain every word of the search text, ignoring case and accents. Words are separated by spaces. The search text SHALL be at most 100 characters.

#### Scenario: Match in the title or the details
- **GIVEN** a report titled «Poste sin luz» and another with «el poste está inclinado» in its details
- **WHEN** a caller searches «poste»
- **THEN** both reports are returned

#### Scenario: Every word must match
- **GIVEN** a report titled «Basura en el parque» and another titled «Basura en la avenida»
- **WHEN** a caller searches «basura parque»
- **THEN** only «Basura en el parque» is returned

#### Scenario: Case and accents are ignored
- **GIVEN** a report titled «Fuga de agua en Jr. Ejemplo»
- **WHEN** a caller searches «FUGA agüa»
- **THEN** that report is returned

#### Scenario: Special characters are plain text
- **WHEN** a caller searches «100%» or «a_b»
- **THEN** only reports that contain that literal text are returned

#### Scenario: Nothing matches
- **WHEN** no report matches the search text
- **THEN** the API answers 200 with an empty list and no next page

#### Scenario: Search text too long
- **WHEN** the search text has more than 100 characters
- **THEN** the API answers 400

#### Scenario: No token
- **WHEN** the request carries no valid token
- **THEN** the API answers 401

### Requirement: The API orders results by recency or by distance
The API SHALL order the results from newest to oldest by default. When the caller asks to order by distance, which needs their location, it SHALL order them from the nearest to the farthest, with ties going to the newest. Every item SHALL carry its distance when the caller sends a location.

#### Scenario: Newest first
- **GIVEN** three matching reports published at 10:00, 11:00 and 12:00
- **WHEN** a caller searches without asking for an order
- **THEN** the results start with the 12:00 report, then 11:00, then 10:00

#### Scenario: Nearest first
- **GIVEN** matching reports 500 m, 80 m and 1200 m from the caller
- **WHEN** the caller searches ordering by distance and sending their location
- **THEN** the results are the 80 m, 500 m and 1200 m reports, in that order, each with its distance

#### Scenario: Order by distance without location
- **WHEN** a caller asks to order by distance without sending latitude and longitude
- **THEN** the API answers 400

#### Scenario: Incomplete or invalid location
- **WHEN** only one of latitude and longitude is sent, or one is out of range
- **THEN** the API answers 400

### Requirement: The API searches by words, by closeness or by both
The API SHALL accept a search with words, a search ordered by distance without words (every report, nearest first), or both. A search without words that is not ordered by distance SHALL be rejected. Text made only of spaces counts as no words.

#### Scenario: Only nearby
- **WHEN** a caller searches without words, ordering by distance and sending their location
- **THEN** every published report is returned, nearest first

#### Scenario: Neither words nor nearby
- **WHEN** a caller searches without words and without ordering by distance
- **THEN** the API answers 400

### Requirement: Search results carry what a card shows
Each result SHALL carry the same data as an item of the feed: id, title, category, status, place reference, publication date, a link to its first photo that expires after one hour, and its distance when the caller sent a location. Results SHALL NOT carry the details, the author name, or any user id.

#### Scenario: The publisher is never revealed
- **WHEN** any caller searches reports
- **THEN** no result carries an author name or the user id of the session that published it

### Requirement: Search results are paginated with a cursor
The API SHALL return the results in pages of at most the requested size (20 by default, between 1 and 50), with an opaque cursor for the next page, or none when there are no more. A next page SHALL continue right after the last result of the previous one, without repeating or skipping any.

#### Scenario: Two pages
- **GIVEN** 25 matching reports
- **WHEN** a caller asks for the first page without a size and then for the page of the cursor it returned
- **THEN** the first page has 20 results and a cursor
- **AND** the second page has the other 5 and no cursor

#### Scenario: Same distance across a page boundary
- **GIVEN** two reports at the same distance that fall on both sides of a page boundary
- **WHEN** the caller walks through the pages ordering by distance
- **THEN** each of them appears exactly once

#### Scenario: Invalid cursor or size
- **WHEN** the cursor is not one the API issued for that order, or the size is below 1 or above 50
- **THEN** the API answers 400

### Requirement: Buscar searches by words
Buscar SHALL show a search box and, when the neighbor submits it, SHALL show the reports that match as the same cards Inicio uses, opening the report when one is tapped. Nothing SHALL be searched until the neighbor submits.

#### Scenario: Search
- **GIVEN** a report titled «Poste sin luz en Jr. Ejemplo»
- **WHEN** the neighbor writes «poste» in Buscar and submits
- **THEN** the card of that report is shown

#### Scenario: Before searching
- **WHEN** the neighbor opens Buscar
- **THEN** the search box and «Cerca de mí» are shown with no results and no request is made

#### Scenario: Open a result
- **WHEN** the neighbor taps the card of a result
- **THEN** the page of that report opens

### Requirement: Buscar orders by distance with «Cerca de mí»
Buscar SHALL offer a «Cerca de mí» option. Turning it on SHALL ask for the device location; once it is given, results SHALL be ordered by distance and their cards SHALL show it. With «Cerca de mí» on, submitting without words SHALL show every report, nearest first.

#### Scenario: Turn on «Cerca de mí»
- **GIVEN** the neighbor searched «basura»
- **WHEN** they turn on «Cerca de mí» and allow the location
- **THEN** the results are shown nearest first, each with its distance

#### Scenario: Only nearby
- **GIVEN** «Cerca de mí» is on and the location was given
- **WHEN** the neighbor submits an empty search
- **THEN** every report is shown, nearest first

#### Scenario: Location declined
- **WHEN** the neighbor turns on «Cerca de mí» and declines, or the position cannot be obtained
- **THEN** a message in Spanish says the location could not be used
- **AND** «Cerca de mí» stays off and the results stay ordered by recency

#### Scenario: Previously shared location
- **GIVEN** the neighbor shared their location before and the browser still allows it
- **WHEN** they open Buscar
- **THEN** the cards show the distance, without asking again and without turning on «Cerca de mí»

### Requirement: Buscar shows empty, loading and error states
Buscar SHALL show a loading state while the results arrive, and a message when nothing matches the search. When the search cannot be done it SHALL show a message in Spanish with «Reintentar». More results SHALL be loaded when the neighbor reaches the end of the list.

#### Scenario: No results
- **WHEN** the neighbor searches «xyz» and nothing matches
- **THEN** a message in Spanish says no report matches «xyz»

#### Scenario: Loading
- **WHEN** the neighbor submits and the results have not arrived yet
- **THEN** a loading state is shown in place of the results

#### Scenario: Search fails and retry
- **GIVEN** the API cannot be reached
- **WHEN** the neighbor submits a search
- **THEN** a message in Spanish says the search could not be done and offers «Reintentar»

#### Scenario: More results
- **GIVEN** more results than fit in the first page
- **WHEN** the neighbor scrolls to the end of the list
- **THEN** the next results appear below without reloading the page

### Requirement: The search is kept in the address
Buscar SHALL keep the submitted words and whether «Cerca de mí» is on in its address, so that going back from a report, reloading, or opening the same address shows the same search again.

#### Scenario: Back from a result
- **GIVEN** the neighbor searched «poste» and opened a result
- **WHEN** they go back
- **THEN** Buscar shows «poste» in the search box and the same results

#### Scenario: Reload
- **GIVEN** the neighbor searched «poste» with «Cerca de mí» on
- **WHEN** they reload the page and the browser still allows the location
- **THEN** the same search is shown, nearest first
