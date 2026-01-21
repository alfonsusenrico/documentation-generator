# GitHub Settings Checklist

Apply these settings after repo creation.

## Branch Protection (main)
- Require pull request before merge.
- Require status checks to pass:
  - CI
- Require branch to be up to date before merge (recommended).
- Disallow force pushes.

## Environments
- Create `staging` environment.
- Create `production` environment and require reviewers for deploy approvals.

## Required Secrets
- `SSH_HOST`
- `SSH_USER`
- `SSH_KEY`
- `SSH_PORT`
- `OPENAI_API_KEY` (for AI automation workflows)

## Optional Variables
- `OPENAI_MODEL` (default: gpt-5-mini)
