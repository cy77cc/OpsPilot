## ADDED Requirements

### Requirement: Frontend production runtime smoke verification MUST detect white-screen regressions
The system MUST include a frontend runtime smoke verification step that exercises the production build in a browser-like environment and detects fatal initialization failures before release.

#### Scenario: Fatal production runtime error
- **WHEN** the frontend production build is loaded in the smoke verification environment
- **THEN** the verification fails if a fatal page or module initialization error occurs
- **AND** the failure is reported before release

### Requirement: Frontend smoke verification MUST prove shell availability independent of AI surface health
The system MUST verify that the main application shell renders even when the AI assistant surface cannot initialize successfully.

#### Scenario: AI surface fails during smoke verification
- **WHEN** the smoke verification simulates or encounters an AI surface initialization failure
- **THEN** the application shell still renders
- **AND** the verification confirms that the failure remains local to the AI surface
