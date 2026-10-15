# app-navigation Specification

## Purpose

Defines how a neighbor moves between the sections of Punto Vecinal (Inicio, Buscar, Reportar, Avisos, Mapa and Perfil) on mobile and on desktop, with the same sections in both.

## Requirements

### Requirement: Mobile bottom bar
On mobile, the app SHALL show a bottom bar with Inicio, Buscar, Reportar, Avisos and Mapa, in that order, with Reportar highlighted as the main action, and SHALL mark the current section.

#### Scenario: Bar on a section
- **GIVEN** the neighbor is on a mobile device on Inicio
- **THEN** the bottom bar shows Inicio, Buscar, Reportar, Avisos and Mapa
- **AND** Inicio is marked as the current section

#### Scenario: Move between sections
- **WHEN** the neighbor taps Buscar, Avisos or Mapa in the bottom bar
- **THEN** that section opens and is marked as current

### Requirement: Mobile top bar and avatar
On mobile, the app SHALL show a top bar with the Punto Vecinal brand and the neighbor's avatar, and the avatar SHALL open Perfil.

#### Scenario: Open Perfil
- **WHEN** the neighbor taps the avatar
- **THEN** Perfil opens

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
- **GIVEN** the neighbor is on Mapa
- **WHEN** the window changes between mobile and desktop width
- **THEN** they stay on Mapa and the navigation switches to the bar of that width

### Requirement: Sections without content show «Próximamente»
Until their own stories land, Buscar, Avisos, Mapa and Perfil SHALL open and show «Próximamente».

#### Scenario: Open a section not built yet
- **WHEN** the neighbor opens Buscar, Avisos, Mapa or Perfil
- **THEN** the section opens with the message «Próximamente»
- **AND** the navigation stays visible

### Requirement: Welcome screen has no navigation
The app SHALL NOT show any navigation bar on the welcome screen.

#### Scenario: Welcome screen
- **WHEN** the neighbor is on the welcome screen
- **THEN** no top bar, bottom bar or header is shown
