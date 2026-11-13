# Spec Delta

## ADDED Requirements

### Requirement: Step 2 collects photos
Step 2, «Foto y lugar», SHALL let the neighbor take a photo or choose one from the device, SHALL accept from 1 up to 3 photos, and SHALL show a thumbnail of each with an option to remove it.

#### Scenario: Add photos
- **GIVEN** the neighbor is on step 2
- **WHEN** they take or choose a photo
- **THEN** a thumbnail of it is shown with an option to remove it

#### Scenario: Remove a photo
- **GIVEN** the neighbor added two photos
- **WHEN** they remove one
- **THEN** only the other thumbnail remains
- **AND** they can add another photo

#### Scenario: Up to three photos
- **GIVEN** the neighbor added 3 photos
- **THEN** three thumbnails are shown and no more photos can be added

#### Scenario: A file that is not an image
- **WHEN** the neighbor chooses a file that cannot be read as an image
- **THEN** a message in Spanish says the photo could not be used
- **AND** the photos already added stay as they were

### Requirement: Step 2 does not accept more than 3 photos
The app SHALL tell the neighbor, in Spanish, when they try to add more photos than the 3 allowed, and SHALL NOT keep the photos that exceed the limit.

#### Scenario: Fourth photo
- **GIVEN** the neighbor already added 3 photos
- **WHEN** they try to add another one
- **THEN** a message in Spanish says that a report takes at most 3 photos
- **AND** the 3 photos stay as they were

#### Scenario: Several photos at once over the limit
- **GIVEN** the neighbor already added 2 photos
- **WHEN** they choose 3 photos at once
- **THEN** only the first of them is added, up to the limit of 3
- **AND** a message in Spanish says that a report takes at most 3 photos

### Requirement: Photos are cleaned and reduced on the device
The app SHALL, before keeping a photo, remove the location of the device and any other hidden data of the file, and SHALL reduce its size so that it uses little mobile data. Photos SHALL NOT be sent to the server while the neighbor is filling the form.

#### Scenario: Photo with device location
- **GIVEN** the neighbor chooses a photo whose file carries the device location
- **WHEN** the photo is kept in the form
- **THEN** the kept photo carries no location nor other hidden file data

#### Scenario: Large photo
- **WHEN** the neighbor chooses a photo larger than the size the app keeps
- **THEN** the kept photo is smaller than the original
- **AND** it still looks like the same picture

#### Scenario: Nothing is uploaded yet
- **WHEN** the neighbor adds, removes or keeps photos in step 2
- **THEN** no photo is sent to the server

### Requirement: Step 2 marks the place on a map
Step 2 SHALL show a map with a pin that the neighbor can move to the exact place of the problem, SHALL keep that location, and SHALL offer an optional text field to write a reference for the place (for example «frente al mercado»).

#### Scenario: Move the pin
- **GIVEN** the neighbor is on step 2
- **WHEN** they move the pin on the map
- **THEN** the pin stays at the new position and that location is the one kept for the report

#### Scenario: Write a reference
- **WHEN** the neighbor writes «frente al mercado» in the reference field
- **THEN** the text is kept together with the location

#### Scenario: Skip the reference
- **GIVEN** the neighbor placed the pin and added a photo
- **WHEN** they leave the reference empty and tap «Seguir»
- **THEN** the form moves to step 3

### Requirement: Step 2 offers to use the device location
Step 2 SHALL offer a «Usar mi ubicación» button and SHALL ask for the location of the device only when the neighbor taps it, never on its own. If they accept, the app SHALL place the pin at that position and keep it as the location. If they decline, or the position cannot be obtained, the app SHALL let them place the pin by hand without blocking the form.

#### Scenario: Nothing is asked on its own
- **WHEN** step 2 opens
- **THEN** the app does not ask for the location of the device
- **AND** the map is shown with the «Usar mi ubicación» button and no location marked

#### Scenario: Accept sharing the location
- **GIVEN** the neighbor is on step 2
- **WHEN** they tap «Usar mi ubicación» and accept to share their location
- **THEN** the pin is placed at their position and that position is kept as the location
- **AND** they can still move the pin

#### Scenario: Decline sharing the location
- **GIVEN** the neighbor is on step 2
- **WHEN** they tap «Usar mi ubicación» and decline to share their location
- **THEN** the map is shown with no location marked and a message in Spanish says they can mark the place by hand
- **AND** moving the pin or tapping the map marks the location

#### Scenario: Position cannot be obtained
- **WHEN** the neighbor taps «Usar mi ubicación» and the device cannot provide a position (no signal, timeout or no support)
- **THEN** a message in Spanish says the position could not be obtained and they can mark the place by hand

#### Scenario: Ask again
- **GIVEN** the neighbor declined or the position failed
- **WHEN** they tap «Usar mi ubicación»
- **THEN** the app tries to get the device position again

#### Scenario: Use the button with a location already marked
- **GIVEN** the neighbor already marked a location
- **WHEN** they tap «Usar mi ubicación» and accept
- **THEN** the pin moves to the device position, replacing the previous location

### Requirement: Step 2 validates photos and place
The app SHALL NOT leave step 2 forward while there is no photo or no location has been marked, and SHALL tell the neighbor in Spanish what is missing. The neighbor SHALL be able to go back to step 1 at any time without losing what they entered.

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
- **GIVEN** the neighbor has 1 to 3 photos and a location
- **WHEN** they tap «Seguir»
- **THEN** the form moves to step 3, «Revisar»

#### Scenario: Go back
- **GIVEN** the neighbor is on step 2 with photos and a location
- **WHEN** they go back to step 1 and tap «Seguir» again
- **THEN** step 2 shows the same photos, location and reference

## MODIFIED Requirements

### Requirement: Closing the form asks what to do with the draft
The app SHALL, when the neighbor taps the close button with something entered (a category, «¿Qué pasa?» or «Más detalles», ignoring whitespace-only text, or a photo, a marked location or a place reference), ask whether to save the draft or discard it, and SHALL take them to Inicio after they choose. If nothing is entered, the app SHALL take them to Inicio without asking.

#### Scenario: Close with data
- **GIVEN** the neighbor wrote part of «¿Qué pasa?»
- **WHEN** they tap the close button
- **THEN** a dialog «¿Guardar borrador?» offers «Guardar borrador» and «Descartar»

#### Scenario: Close with only step 2 data
- **GIVEN** the neighbor added a photo but wrote nothing in step 1
- **WHEN** they tap the close button
- **THEN** the dialog «¿Guardar borrador?» is shown

#### Scenario: Save the draft
- **GIVEN** the close dialog is open
- **WHEN** the neighbor chooses «Guardar borrador»
- **THEN** Inicio opens
- **AND** opening Reportar again shows what they wrote

#### Scenario: Discard the draft
- **GIVEN** the close dialog is open
- **WHEN** the neighbor chooses «Descartar»
- **THEN** Inicio opens
- **AND** opening Reportar again shows an empty form on step 1 without photos or location

#### Scenario: Dismiss the dialog
- **GIVEN** the close dialog is open
- **WHEN** the neighbor dismisses it without choosing
- **THEN** the form stays open with what they wrote

#### Scenario: Close an empty form
- **GIVEN** nothing is written
- **WHEN** the neighbor taps the close button
- **THEN** Inicio opens without a dialog

### Requirement: Draft is kept when leaving the form
The app SHALL keep what the neighbor entered in the Reportar form on their device (including photos, location and place reference), and SHALL restore it when they open the form again.

#### Scenario: Leave and come back
- **GIVEN** the neighbor chose a category and wrote part of «¿Qué pasa?»
- **WHEN** they leave the form without closing it (for example switching tab or reloading) and open Reportar again
- **THEN** the category and the text are restored as they left them

#### Scenario: Photos, location and reference are restored
- **GIVEN** the neighbor added two photos, moved the pin and wrote a reference
- **WHEN** they leave the form without closing it and open Reportar again
- **THEN** step 2 shows the same two photos, the pin at the same place and the same reference

#### Scenario: Close and reopen the app
- **GIVEN** the neighbor left a draft and closed the app
- **WHEN** they reopen the app on the same device and open Reportar
- **THEN** the draft is restored

#### Scenario: Storage unavailable
- **WHEN** the device does not allow storing the draft
- **THEN** the form still works, without restoring the draft later

#### Scenario: Photos cannot be stored
- **GIVEN** the device does not allow storing the photos but allows storing the text
- **WHEN** the neighbor comes back to the form
- **THEN** the text, location and reference are restored
- **AND** the photos are not, so step 2 asks for them again
