# Spec Delta

## ADDED Requirements

### Requirement: Mapa opens the map section
The bottom bar and the desktop header SHALL show Mapa as an available section. Tapping it SHALL open Mapa and mark it as the current section, and opening the Mapa address directly SHALL open Mapa too.

#### Scenario: Tap Mapa on mobile
- **GIVEN** the neighbor is on a mobile device on Inicio
- **WHEN** they tap Mapa in the bottom bar
- **THEN** Mapa opens and is marked as the current section

#### Scenario: Mapa on desktop
- **GIVEN** the neighbor is on desktop on Inicio
- **WHEN** they click Mapa in the header
- **THEN** Mapa opens and is marked as the current section

#### Scenario: Open the map address
- **WHEN** the neighbor opens the Mapa address from a link or by typing it
- **THEN** Mapa opens

## REMOVED Requirements

### Requirement: Mapa is shown as not available
**Reason**: The map exists now (HU-06, PV-15), so Mapa is no longer marked as not available and no «Próximamente» notice is shown for it.
**Migration**: Mapa is a normal section; see «Mapa opens the map section».

### Requirement: The map address opens Inicio
**Reason**: The Mapa address now opens the map section instead of redirecting to Inicio.
**Migration**: See «Mapa opens the map section».
