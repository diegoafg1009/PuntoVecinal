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
The app SHALL, when the neighbor taps the close button with something written (a category, «¿Qué pasa?» or «Más detalles», ignoring whitespace-only text), ask whether to save the draft or discard it, and SHALL take them to Inicio after they choose. If nothing is written, the app SHALL take them to Inicio without asking.

#### Scenario: Close with data
- **GIVEN** the neighbor wrote part of «¿Qué pasa?»
- **WHEN** they tap the close button
- **THEN** a dialog «¿Guardar borrador?» offers «Guardar borrador» and «Descartar»

#### Scenario: Save the draft
- **GIVEN** the close dialog is open
- **WHEN** the neighbor chooses «Guardar borrador»
- **THEN** Inicio opens
- **AND** opening Reportar again shows what they wrote

#### Scenario: Discard the draft
- **GIVEN** the close dialog is open
- **WHEN** the neighbor chooses «Descartar»
- **THEN** Inicio opens
- **AND** opening Reportar again shows an empty form on step 1

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
The app SHALL keep what the neighbor wrote in the Reportar form on their device, and SHALL restore it when they open the form again.

#### Scenario: Leave and come back
- **GIVEN** the neighbor chose a category and wrote part of «¿Qué pasa?»
- **WHEN** they leave the form without closing it (for example switching tab or reloading) and open Reportar again
- **THEN** the category and the text are restored as they left them

#### Scenario: Close and reopen the app
- **GIVEN** the neighbor left a draft and closed the app
- **WHEN** they reopen the app on the same device and open Reportar
- **THEN** the draft is restored

#### Scenario: Storage unavailable
- **WHEN** the device does not allow storing the draft
- **THEN** the form still works, without restoring the draft later

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
