# Spec Delta

## ADDED Requirements

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
