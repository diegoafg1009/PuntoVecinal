# Spec Delta

## MODIFIED Requirements

### Requirement: Mobile bottom bar
On mobile, the app SHALL show a bottom bar with Inicio, Buscar, Reportar, Avisos and Mapa, in that order, with Reportar highlighted as the main action, and SHALL mark the current section.

#### Scenario: Bar on a section
- **GIVEN** the neighbor is on a mobile device on Inicio
- **THEN** the bottom bar shows Inicio, Buscar, Reportar, Avisos and Mapa
- **AND** Inicio is marked as the current section

#### Scenario: Move between sections
- **WHEN** the neighbor taps Buscar or Avisos in the bottom bar
- **THEN** that section opens and is marked as current

### Requirement: Desktop header
On desktop, the app SHALL show a header with the Punto Vecinal brand, the menu Inicio, Buscar, Avisos and Mapa, the «Reportar» button and the avatar, and SHALL mark the current section. The content SHALL be centered.

#### Scenario: Header on desktop
- **GIVEN** the neighbor is on desktop on Inicio
- **THEN** the header shows the brand, Inicio, Buscar, Avisos, Mapa, the «Reportar» button and the avatar
- **AND** Inicio is marked as the current section

#### Scenario: Avatar opens Perfil on desktop
- **WHEN** the neighbor clicks the avatar
- **THEN** Perfil opens

### Requirement: Same sections on mobile and desktop
The app SHALL offer the same sections on mobile and on desktop, and the same address SHALL open the same section on both.

#### Scenario: Resize
- **GIVEN** the neighbor is on Avisos
- **WHEN** the window changes between mobile and desktop width
- **THEN** they stay on Avisos and the navigation switches to the bar of that width

### Requirement: Sections without content show «Próximamente»
Until its own story lands, Perfil SHALL open and show «Próximamente».

#### Scenario: Open a section not built yet
- **WHEN** the neighbor opens Perfil
- **THEN** the section opens with the message «Próximamente»
- **AND** the navigation stays visible

## ADDED Requirements

### Requirement: Mapa is shown as not available
Until the map exists, the bottom bar and the desktop header SHALL show Mapa marked as not available. Tapping it SHALL show a «Próximamente» notice and SHALL keep the neighbor on the screen where they were, with that screen unchanged.

#### Scenario: Mapa is marked
- **WHEN** the neighbor looks at the bottom bar on mobile, or at the header on desktop, on any screen
- **THEN** Mapa is shown and marked as not available

#### Scenario: Tap Mapa
- **GIVEN** the neighbor is on Buscar with results shown
- **WHEN** they tap Mapa
- **THEN** a notice says «Próximamente»
- **AND** they stay on Buscar with the same results, and Buscar is still marked as the current section

### Requirement: The map address opens Inicio
Opening the address of Mapa directly SHALL take the neighbor to Inicio, since the map section does not exist yet.

#### Scenario: Open the map address
- **WHEN** the neighbor opens the Mapa address from a link or by typing it
- **THEN** Inicio opens
