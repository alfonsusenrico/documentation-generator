# Quality Standard - {{PROJECT_NAME}}

**Tier:** 1 (Minimum Effective Standard)  
**Client/Owner:** {{CLIENT_OR_OWNER}}

---

## 1) Purpose
{{QUALITY_PURPOSE}}

## 2) CI Quality Gates (Required)
| Gate | Command | Required |
|------|---------|----------|
| Lint | {{LINT_CMD}} | Yes |
| Type-check | {{TYPE_CMD}} | Yes |
| Unit tests | {{TEST_CMD}} | Yes |
| Build | {{BUILD_CMD}} | Yes |
| Dependency audit | {{DEP_AUDIT_CMD}} | Yes |

Container scan runs in CI and fails on high/critical issues.

## 3) Definition of Done
- CI green (lint, type, tests, build, security).
- .env.example updated if env vars change.
- Docs updated if behavior changes.
- Health endpoint returns status + version + uptime.
- Rollback plan documented for risky changes.
- {{DOD_PROJECT_SPECIFIC}}

## 4) Environment and Docs Hygiene
- .env is never committed; only .env.example.
- Staging and production use separate .env files.
- Architecture doc includes ports, env vars, dependencies, health endpoint.

## 5) Health Contract
- Endpoint: {{HEALTH_ENDPOINT}}
- Must return: status OK, version/commit hash, uptime.
- {{HEALTH_NOTES}}

## 6) Change Risk Notes
{{RISK_NOTES}}
