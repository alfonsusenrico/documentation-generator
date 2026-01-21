# Deployment Standard - {{PROJECT_NAME}}

**Tier:** 1 (Minimum Effective Standard)  
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

## 4) Server Layout
- Staging: {{DEPLOY_PATH_STAGING}}
- Production: {{DEPLOY_PATH_PROD}}

Each environment directory contains:
- compose file (compose.staging.yml or compose.prod.yml)
- .env
- volumes/ (if needed)

## 5) Routing (Reverse Proxy)
- Staging domain: {{STAGING_DOMAIN}}
- Production domain: {{PROD_DOMAIN}}
- Proxy routes by hostname to the correct compose project.

## 6) Image Tag Policy
- Deploy uses immutable tag: sha-<commit>.
- .env fields:
  - IMAGE_NAME={{IMAGE_NAME}}
  - IMAGE_TAG=sha-<commit>

## 7) Staging Deploy (from dev)
1) Update IMAGE_TAG in staging .env.
2) docker compose pull
3) docker compose up -d
4) Health check: curl -fsS http://localhost:{{APP_PORT}}{{HEALTH_ENDPOINT}}

## 8) Production Deploy (from main, approval gated)
1) Update IMAGE_TAG in prod .env.
2) docker compose pull
3) docker compose up -d
4) Health check: curl -fsS http://localhost:{{APP_PORT}}{{HEALTH_ENDPOINT}}
5) On success, write IMAGE_TAG to .last_good.

## 9) Rollback
1) Set IMAGE_TAG to .last_good.
2) docker compose up -d
3) Health check.

## 10) Deploy Records
- Record each deploy to {{DEPLOY_LOG_PATH}} with:
  - commit sha, time, environment, result, workflow link.
- Metrics JSON log: {{METRICS_LOG_PATH}}
