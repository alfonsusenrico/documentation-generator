# Deployment Standard - {{PROJECT_NAME}}

**Tier:** 2 (Production System)  
**Service:** {{SERVICE_NAME}}  
**Image:** {{IMAGE_NAME}}

---

## 1) Environments
- local: developer machine
- staging: rehearsal environment
- production: live environment

## 2) Server Requirements
- Docker installed
- docker compose plugin available
- deploy user with docker permissions
- SSH key-based auth

## 3) Registry Access
- docker login performed on server for private registries
- tokens stored securely (avoid embedding in compose files)

## 4) Isolation Rules (Mandatory)
- Separate domains, directories, env files, and storage.
- Separate databases (recommended).
- No shared volumes between staging and production.

## 5) Pre-Deploy Gates
- CI green (lint, type, tests, build, security, integration).
- Migration plan reviewed for schema changes.
- Rollback tag verified (.last_good).

## 6) Staging Deploy (from dev)
1) Update IMAGE_TAG in staging .env.
2) docker compose pull
3) docker compose up -d
4) Run integration smoke checks.
5) Health check: curl -fsS http://localhost:{{APP_PORT}}{{HEALTH_ENDPOINT}}

## 7) Production Deploy (from main, approval gated)
1) Update IMAGE_TAG in prod .env.
2) docker compose pull
3) docker compose up -d
4) Apply migration step if needed: {{MIGRATION_STEP}}
5) Health check: curl -fsS http://localhost:{{APP_PORT}}{{HEALTH_ENDPOINT}}
6) On success, write IMAGE_TAG to .last_good.

## 8) Rollback
1) Set IMAGE_TAG to .last_good.
2) docker compose up -d
3) Health check.

## 9) Concurrency Controls
- Only one deploy per environment at a time (workflow concurrency).

## 10) Observability
- Alerts: {{ALERTING_NOTES}}
- Dashboards: {{DASHBOARD_LINKS}}

## 11) Deploy Records
- Record each deploy to {{DEPLOY_LOG_PATH}} with:
  - commit sha, time, environment, result, workflow link.
- Metrics JSON log: {{METRICS_LOG_PATH}}
