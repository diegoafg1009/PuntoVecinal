# app-entry Specification

## Purpose

Defines how a neighbor enters Punto Vecinal for the first time and on later visits: a welcome screen with guest entry through an anonymous session, with no data requested.

## Requirements

### Requirement: Welcome screen on first visit
The app SHALL show a welcome screen with the options «Comenzar» and «Ya tengo una cuenta» when the neighbor opens the app on a device where no entry option has been chosen yet.

#### Scenario: First visit
- **WHEN** a neighbor opens the app for the first time on a device
- **THEN** the welcome screen is shown with exactly the two options «Comenzar» and «Ya tengo una cuenta»

### Requirement: Guest entry creates an anonymous session
The app SHALL create an anonymous session on the device, without asking for any data, when the neighbor chooses «Comenzar», and SHALL then enter the app.

#### Scenario: Continue as guest
- **WHEN** the neighbor chooses «Comenzar»
- **THEN** an anonymous session is created on the device without any form or data request
- **AND** the neighbor enters the app

#### Scenario: Anonymous session cannot be created
- **WHEN** the neighbor chooses «Comenzar» and the session cannot be created
- **THEN** the neighbor stays on the welcome screen and sees an error message in Spanish with the option to try again

### Requirement: Returning visits skip the welcome screen
The app SHALL enter directly, without showing the welcome screen, when the neighbor reopens the app on a device where an entry option was already chosen.

#### Scenario: Reopen after choosing guest
- **GIVEN** the neighbor previously continued as guest on this device
- **WHEN** they reopen the app
- **THEN** they enter the app directly without seeing the welcome screen

### Requirement: Anonymous session persists on the device
The app SHALL keep the neighbor in the same anonymous session when they return from the same device.

#### Scenario: Same device, same session
- **GIVEN** the neighbor entered as guest and the API identified them with user id X
- **WHEN** they reopen the app from the same device
- **THEN** the API identifies them with the same user id X, with the token refreshed automatically if it expired

### Requirement: Token refresh keeps the same anonymous user
The app SHALL keep the neighbor as the same anonymous user when their session token is refreshed, for as long as the device keeps the session. Sessions of anonymous users SHALL NOT expire by time limit.

#### Scenario: Refresh after the token expires
- **GIVEN** the neighbor entered as guest with user id X and their access token has expired
- **WHEN** the app refreshes the session
- **THEN** the API identifies them with the same user id X

#### Scenario: Device loses the session
- **GIVEN** the neighbor entered as guest
- **WHEN** the device's stored session is lost (for example, site data cleared)
- **THEN** the app shows the welcome screen and a new guest entry creates a different anonymous user

### Requirement: Login is not available yet
The app SHALL show «Próximamente» when the neighbor taps «Ya tengo una cuenta» while login is not available, and SHALL let them continue as guest.

#### Scenario: Tap Ya tengo una cuenta
- **WHEN** the neighbor taps «Ya tengo una cuenta»
- **THEN** the message «Próximamente» is shown
- **AND** the neighbor can still choose «Comenzar»
- **AND** no entry option is remembered by this tap alone
