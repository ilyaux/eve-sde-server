# Changelog

All notable changes to this project will be documented here.

This project follows a pragmatic changelog format inspired by
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions are not yet
published; entries currently track unreleased work on `main`.

## Unreleased

### Added

- REST taxonomy endpoints for categories and groups.
- GraphQL taxonomy fields for categories and groups.
- Go SDK taxonomy methods.
- CI coverage for server vet/tests, race tests, SDK checks, binary build, and
  Docker image build.
- Community contribution, security, issue, and pull request templates.

### Changed

- API keys are stored as SHA-256 hashes at rest. Raw keys are only returned when
  created.
- Admin key management now uses the auth manager instead of direct database
  writes.
- Docker startup initializes the SQLite schema before starting the server.

### Fixed

- Admin dashboard remains reachable when API-key authentication is enabled.
- Admin key list rendering treats key names as text to avoid HTML injection.
- SDE changelog item counts use the current database schema.
- SDE import history is recorded after successful imports.
