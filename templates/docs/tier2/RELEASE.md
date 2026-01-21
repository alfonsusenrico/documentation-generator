# Release Process - {{PROJECT_NAME}}

**Tier:** 2 (Production System)  
**Repo:** {{REPO_URL}}  
**Image:** {{IMAGE_NAME}}

---

## 1) Branching Flow
- Daily work merges into dev.
- Release is PR dev -> main (squash merge recommended).
- main changes only via PR.

## 2) CI Gate
- Required checks: lint, type, tests, build, security, integration tests.
- All gates must pass before merge.

## 3) Artifact Tagging
- Primary deploy unit: sha-<commit> tag.
- Optional convenience tag: latest (do not rely on it for rollback).

## 4) Release Notes
- Draft generated from PR content and commit messages.
- Must include migration notes and rollback notes.
- {{MIGRATION_NOTES}}
- {{ROLLBACK_NOTES}}

## 5) Deploy Flow
- Staging deploy: push to dev.
- Production deploy: push to main, approval gated, concurrency controlled.

## 6) Rollback
- Use .last_good tag and re-run compose up.
