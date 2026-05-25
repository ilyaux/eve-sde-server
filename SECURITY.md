# Security Policy

## Supported Versions

Security fixes target the `main` branch. Until the project starts publishing
versioned releases, users should deploy from the latest successful `main` CI run.

## Reporting a Vulnerability

Please do not file public GitHub issues for vulnerabilities.

Use GitHub private vulnerability reporting when available for this repository. If
that is unavailable, contact the maintainer privately and include:

- A short description of the issue.
- Affected endpoints, commands, or deployment modes.
- Steps to reproduce.
- Impact and any known mitigations.

Do not include live secrets, production API keys, or private EVE/ESI data in the
report.

## Security Expectations

- API keys are stored as SHA-256 hashes at rest.
- Raw API keys are returned only once, when created.
- Admin routes use HTTP Basic Auth and should be served behind TLS in public
  deployments.
- Public deployments should set explicit `ALLOWED_ORIGINS` and non-default
  `ADMIN_USERNAME` / `ADMIN_PASSWORD` values.

## Disclosure

The maintainer will acknowledge valid reports as soon as practical, coordinate a
fix, and credit reporters when requested and appropriate.
