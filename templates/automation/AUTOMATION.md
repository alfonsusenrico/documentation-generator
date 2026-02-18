# Automation Profile

- Automation level: `{{AUTOMATION_LEVEL}}`
- Deploy mode: `{{DEPLOY_MODE}}`

## CI Commands
- Install: `{{INSTALL_CMD}}`
- Lint: `{{LINT_CMD}}`
- Type check: `{{TYPE_CMD}}`
- Test: `{{TEST_CMD}}`
- Integration test: `{{INTEGRATION_TEST_CMD}}`
- Build: `{{BUILD_CMD}}`
- Security scan: `{{DEP_AUDIT_CMD}}`

## Behavior by Level
- `repo_only`: scaffold + docs only, no CI/CD workflows.
- `repo_ci`: scaffold + CI workflows, no deploy workflows.
- `repo_ci_cd`: scaffold + CI + CD workflows (when deploy mode is enabled).

## Artifact
- Type: `{{ARTIFACT_TYPE}}`
- Image: `{{IMAGE_NAME}}`
- Path: `{{ARTIFACT_PATH}}`

## Deploy
- If deploy mode is `none`, deploy workflows are not generated.
- If deploy mode is `ssh_compose`, staging/production workflows are generated in manual-first mode (workflow_dispatch) and environment protection is still required.

## Required Manual Configuration
- Repository branch protection rules.
- Environment protection rules for production approvals.
- Secrets for deployment mode (if any).
