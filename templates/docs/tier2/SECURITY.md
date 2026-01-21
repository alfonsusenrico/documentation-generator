# Security - {{PROJECT_NAME}}

**Tier:** 2 (Production System)

---

## 1) Secrets Handling
- Secrets are stored in environment variables or a secrets manager.
- .env is never committed; only .env.example.
- GitHub secret scanning enabled.
- {{SECRETS_NOTES}}

## 2) Dependency Updates
- Dependabot: weekly PRs.
- CI must pass on dependency update PRs.

## 3) Vulnerability Scanning
- Dependency audit: {{DEP_AUDIT_CMD}}
- Container scan: Trivy (fail on high/critical).

## 4) Access Control
- {{ACCESS_CONTROL_NOTES}}

## 5) Incident Response
- See docs/RUNBOOK.md for incident handling.
- {{INCIDENT_RESPONSE_NOTES}}
