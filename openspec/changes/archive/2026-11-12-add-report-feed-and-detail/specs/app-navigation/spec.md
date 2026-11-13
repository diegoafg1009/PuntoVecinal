# Spec Delta

## MODIFIED Requirements

### Requirement: Mobile top bar and avatar
On mobile, the app SHALL show a top bar with the Punto Vecinal brand and the neighbor's avatar, and the avatar SHALL open Perfil. Screens that have their own top bar (the report page) SHALL show it instead of this one, and keep the bottom bar.

#### Scenario: Open Perfil
- **WHEN** the neighbor taps the avatar
- **THEN** Perfil opens

#### Scenario: Screen with its own top bar
- **GIVEN** the neighbor is on a mobile device
- **WHEN** they open a report page
- **THEN** the app top bar with brand and avatar is not shown
- **AND** the bottom bar is shown
