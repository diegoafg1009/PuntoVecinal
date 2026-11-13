# notices Specification

## Purpose
Defines the Avisos section, where neighbors will follow news about the reports they care about. For now it only covers what a neighbor without an account sees there (HU-22, PV-31).

## Requirements

### Requirement: Avisos explains that it needs an account
For a neighbor without an account, Avisos SHALL show, in place of a list, a message saying that notices require an account and that creating accounts will be available soon. It SHALL NOT offer any button or link to sign up or sign in.

#### Scenario: Open Avisos without an account
- **GIVEN** the neighbor entered without an account
- **WHEN** they open Avisos
- **THEN** a message says that notices require an account
- **AND** it says that creating accounts will be available soon
- **AND** no list of notices is shown

#### Scenario: No way to a screen that does not exist
- **GIVEN** the neighbor without an account is on Avisos
- **WHEN** they read the message
- **THEN** no button or link offers to create an account or sign in

#### Scenario: Navigation keeps working
- **GIVEN** the neighbor is on Avisos
- **WHEN** they move to another section from the navigation
- **THEN** that section opens as it does from any other section
