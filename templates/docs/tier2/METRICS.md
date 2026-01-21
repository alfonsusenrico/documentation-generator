# Metrics - {{PROJECT_NAME}}

**Tier:** 2 (Production System)

---

## 1) Required Metrics
- Lead time: PR merge to deploy.
- Change failure rate: deploys needing rollback/fix-forward.
- MTTR: time to restore service after failure.
- Deploy frequency: deploys per week.
- CI pass rate: % green runs.

## 2) Data Sources
- Deploy log: {{DEPLOY_LOG_PATH}}
- Metrics log: {{METRICS_LOG_PATH}}
- CI runs: GitHub Actions
- Alerts/dashboards: {{ALERTING_NOTES}}

## 3) Collection Notes
{{METRICS_COLLECTION_NOTES}}

## 4) Weekly Digest
{{METRICS_DIGEST_NOTES}}
