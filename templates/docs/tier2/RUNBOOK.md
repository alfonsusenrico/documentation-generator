# Runbook - {{PROJECT_NAME}}

**Tier:** 2 (Production System)  
**Service:** {{SERVICE_NAME}}  
**Health endpoint:** {{HEALTH_ENDPOINT}}

---

## 1) Severity Levels
- SEV1: Complete outage or data loss. Immediate response required.
- SEV2: Major feature degradation. Response within hours.
- SEV3: Minor issues or bugs. Fix in normal cycle.

## 2) Detection
- Health check monitor: {{HEALTH_CHECK_MONITOR}}
- Logs location: {{LOG_LOCATION}}
- Alerts channel: {{ALERT_CHANNEL}}
- Dashboards: {{DASHBOARD_LINKS}}

## 3) On-Call & Escalation
- Primary on-call: {{ONCALL_PRIMARY}}
- Escalation path: {{ESCALATION_PATH}}

## 4) First Response Checklist
- Confirm current version: {{VERSION_CHECK}}
- Verify health endpoint responds.
- Review last deploy record: {{DEPLOY_LOG_PATH}}
- Check error logs and metrics.

## 5) Rollback
1) Set IMAGE_TAG to .last_good.
2) docker compose up -d
3) Re-check health endpoint.

## 6) Data / Migration Incidents
{{DATA_INCIDENT_PROCEDURE}}

## 7) Communication
- Stakeholders: {{STAKEHOLDERS}}
- Status updates: {{STATUS_CHANNEL}}

## 8) Post-Incident
- Postmortem doc: {{POSTMORTEM_TEMPLATE}}
- Track follow-up actions and due dates.
