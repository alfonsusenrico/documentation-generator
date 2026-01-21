# Architecture - {{PROJECT_NAME}}

**Tier:** 2 (Production System)  
**Service:** {{SERVICE_NAME}}

---

## 1) Overview
{{ARCH_OVERVIEW}}

## 2) Components
{{ARCH_COMPONENTS}}

## 3) Data Stores
{{DATA_STORES}}

## 4) External Integrations
{{EXTERNAL_DEPENDENCIES}}

## 5) Data Flow
{{ARCH_DATA_FLOW}}

## 6) Environment & Ports
- App port: {{APP_PORT}}
- Health endpoint: {{HEALTH_ENDPOINT}}
- Env vars: see docs/ENV_VARS.md (if present)

## 7) Scaling & Reliability Notes
{{SCALING_NOTES}}

## 8) Observability
- Logging format: {{LOG_FORMAT}}
- Health check: {{HEALTH_ENDPOINT}}
- {{OBS_NOTES}}
