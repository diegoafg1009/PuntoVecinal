# report-creation Specification

## Purpose

Defines the Reportar flow a neighbor follows to report a neighborhood problem: its entry point, the 3-step form and the data each step collects. This change covers the 3-step form and its first step.

## Requirements

### Requirement: Reportar opens a 3-step form
The app SHALL open the Reportar form when the neighbor taps «Reportar» (the highlighted button of the mobile bottom bar or the «Reportar» button of the desktop header) and SHALL show a step indicator with the steps «Qué pasa», «Foto y lugar» and «Revisar», with the current step highlighted and the counter «1 de 3», «2 de 3» or «3 de 3».

#### Scenario: Open the form
- **WHEN** the neighbor taps «Reportar»
- **THEN** the form opens on step 1, «Qué pasa», with the title «Nuevo reporte» and the counter «1 de 3»
- **AND** the indicator shows the three steps «Qué pasa», «Foto y lugar» and «Revisar»

#### Scenario: Mobile form is a focused screen
- **GIVEN** the neighbor is on a mobile device
- **WHEN** the Reportar form is open
- **THEN** the bottom bar and the app top bar are not shown
- **AND** a close button is shown at the top of the form

#### Scenario: Desktop form keeps the header
- **GIVEN** the neighbor is on desktop
- **WHEN** the Reportar form is open
- **THEN** the form is shown as a centered card under the header

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

### Requirement: Step 1 shows the emergency notice
Step 1 SHALL always show a notice that this app does not handle emergencies and that for an emergency or a safety issue the neighbor should call Serenazgo or 105.

#### Scenario: Notice on step 1
- **WHEN** the neighbor is on step 1
- **THEN** the notice «¿Es una emergencia o un tema de seguridad? Llama a Serenazgo o al 105. Esta app no atiende urgencias.» is shown

### Requirement: Step 1 collects category and description
Step 1 SHALL ask the neighbor to choose exactly one category from the fixed list (Limpieza, Alumbrado, Pistas y veredas, Agua y desagüe, Otro) and to write «¿Qué pasa?» as a one-line title, with the help text «En una línea. Es el título que verán los vecinos.», and SHALL offer an optional «Más detalles» field.

#### Scenario: Complete step 1
- **GIVEN** the neighbor is on step 1
- **WHEN** they choose a category, write «¿Qué pasa?» and tap «Seguir»
- **THEN** the form moves to step 2, «Foto y lugar»

#### Scenario: Skip the optional details
- **GIVEN** the neighbor chose a category and wrote «¿Qué pasa?»
- **WHEN** they leave «Más detalles» empty and tap «Seguir»
- **THEN** the form moves to step 2

#### Scenario: Only the fixed categories are offered
- **WHEN** the neighbor looks at the category choice
- **THEN** exactly Limpieza, Alumbrado, Pistas y veredas, Agua y desagüe and Otro are offered

#### Scenario: One category at a time
- **GIVEN** the neighbor chose Limpieza
- **WHEN** they choose Alumbrado
- **THEN** Alumbrado is selected and Limpieza is not

### Requirement: Step 1 validates required fields
The app SHALL NOT leave step 1 while the category is not chosen or «¿Qué pasa?» is empty (or only whitespace), and SHALL tell the neighbor what is missing.

#### Scenario: Empty «¿Qué pasa?»
- **GIVEN** the neighbor is on step 1
- **WHEN** «¿Qué pasa?» is empty and they tap «Seguir»
- **THEN** a message in Spanish shows that «¿Qué pasa?» is missing
- **AND** the form stays on step 1

#### Scenario: No category
- **GIVEN** the neighbor wrote «¿Qué pasa?» but chose no category
- **WHEN** they tap «Seguir»
- **THEN** a message in Spanish shows that the category is missing
- **AND** the form stays on step 1

#### Scenario: Whitespace only
- **WHEN** «¿Qué pasa?» contains only spaces and the neighbor taps «Seguir»
- **THEN** it is treated as empty and the form stays on step 1

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

### Requirement: Category choices come from the API
Step 1 SHALL offer as category choices exactly the categories returned by the API, in the order the API gives them, and SHALL show that the categories are loading until they arrive.

#### Scenario: Categories loaded
- **WHEN** step 1 opens and the API returns the categories
- **THEN** the neighbor sees one choice per category, in the API order, with the API labels

#### Scenario: Categories loading
- **WHEN** step 1 opens and the categories have not arrived yet
- **THEN** a loading state is shown in place of the choices
- **AND** the neighbor cannot continue to step 2 yet

### Requirement: Categories that cannot be loaded can be retried
Step 1 SHALL tell the neighbor, in Spanish, when the categories cannot be loaded, SHALL NOT let them continue without a category, and SHALL offer a way to try again.

#### Scenario: Load failure and retry
- **GIVEN** the API cannot be reached when step 1 opens
- **THEN** a message in Spanish says the categories could not be loaded and offers «Reintentar»
- **WHEN** the neighbor taps «Reintentar» and the API answers
- **THEN** the category choices are shown

#### Scenario: Stored category is not enough without the list
- **GIVEN** a restored draft with a category, a title and details, and the API cannot be reached
- **WHEN** the neighbor taps «Seguir»
- **THEN** the form stays on step 1 with a message in Spanish asking to load the categories again
- **AND** the message disappears once the categories are loaded

### Requirement: Draft drops a category that is no longer offered
When the app restores a draft whose category is not among the categories returned by the API, it SHALL clear that category and keep the rest of the draft.

#### Scenario: Stored category no longer exists
- **GIVEN** a stored draft with a category the API does not return, a title and details
- **WHEN** the neighbor opens Reportar and the categories load
- **THEN** no category is selected
- **AND** the title and details are restored as they were

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
