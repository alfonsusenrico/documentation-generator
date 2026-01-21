# Release Process - {{PROJECT_NAME}}

**Tier:** 1 (Minimum Effective Standard)  
**Repo:** {{REPO_URL}}  
**Image:** {{IMAGE_NAME}}

---

## 1) Branching Flow
- Daily work merges into dev.
- Release is PR dev -> main (squash merge recommended).
- main changes only via PR.

## 2) CI Gate
- Required checks: lint, type, tests, build, security.
- All gates must pass before merge.

## 3) Artifact Tagging
- Primary deploy unit: sha-<commit> tag.
- Optional convenience tag: latest (do not rely on it for rollback).

## 4) Release Notes
- Draft generated from PR content and commit messages.
- Human review before publishing.

## 5) Deploy Flow
- Staging deploy: push to dev.
- Production deploy: push to main, approval gated.

## 6) Rollback
- Use .last_good tag and re-run compose up.
