# Documentation Generator

![Status](https://img.shields.io/badge/status-in--development-blue)

## Summary
Documentation Generator is a private, local-first tool that turns project notes into structured proposals and then initializes a ready-to-use GitHub repository with a full professional scaffold (docs, workflows, CI/CD, compose). It is designed for a single developer workflow and runs on a home server.

## Core Capabilities
- Generate a proposal from a free-text plan using a fixed template (Markdown + optional PDF).
- Edit proposals in-place, save updates, and reuse past proposals without re-generating.
- Initiate a project: create `~/project/<client>/<project>`, generate docs/workflows, init git, and push to GitHub via `gh`.
- Tiered DevOps scaffolding (Tier 1/Tier 2) with CI gates, releases, and staging/production deploys.
- AI automation in generated repos (PR summary + risk, CI failure explainer, release notes draft).
- Deploy metrics logging (JSONL) plus human-readable deploy log.
- Preflight checks to fail early before spending AI tokens.

## Workflow
1) Paste a project plan and generate a proposal.
2) Review and edit the proposal in the UI.
3) Initiate the project to create the scaffolded repo.
4) Configure GitHub settings/secrets and begin implementation.

## Requirements
- Go 1.22+
- Git
- GitHub CLI (`gh`) with authentication
- systemd user services
- Optional (for PDF): `pandoc` + LaTeX (`xelatex`)
- Optional (remote access): cloudflared

## Setup
1) Clone
```bash
git clone https://github.com/alfonsusenrico/documentation-generator.git
cd documentation-generator
```

2) Configure environment
```bash
cp .env.example .env
```
Fill in the values in `.env`.

3) Install and run as a service
```bash
bash start-server
```

4) Open the UI
- Local: `http://localhost:3000`
- Remote: use your Cloudflared tunnel URL

## Environment Variables
| Name | Required | Example | Notes |
|------|----------|---------|-------|
| OPENAI_API_KEY | yes | sk-... | OpenAI API key |
| OPENAI_MODEL | no | gpt-5-mini | Default model |
| PORT | no | 3000 | Server port |
| PUBLIC_BASE_URL | no | https://docgen.example.com | Used for download links |
| LANG | no | en | Output language for body content |
| PREPARED_BY | no | Alfonsus Enrico | Proposal header default |
| ENABLE_PDF | no | 1 | Enable PDF output via pandoc |
| INIT_TOKEN | no | some-token | Optional header token for `/api/init` |
| AGENT_TOKEN | no | some-token | Required to enable `/api/agent/*` endpoints (use header `X-Agent-Token`) |
| GITHUB_OWNER | no | alfonsusenrico | GitHub owner/org slug |
| GIT_USER_NAME | no | Enrico | Optional git identity override |
| GIT_USER_EMAIL | no | you@example.com | Optional git identity override |

## Usage
### Generate a proposal
- Fill in project name, client/owner, and the plan.
- Click **Generate proposal** to produce Markdown (and PDF if enabled).
- Edit directly in the preview and click **Save edits** to persist and update the PDF.

### Load existing proposals
- Use **Existing Proposals** to load from `out/` without re-generating.

### Initiate a project
- Select stack (python/node), tier (1/2), visibility, automation level, and deploy mode.
- Click **Initiate project**.
- Preflight checks validate `gh` auth, git identity, repo name, and target path.
- The system creates `~/project/<client>/<project>`, commits with `[skip ci]`, and pushes `main` + `dev` branches.

### AI automation in generated repos
- Set `OPENAI_API_KEY` (and optional `OPENAI_MODEL`) in repo secrets/vars to enable:
  - PR summary + risk classification
  - CI failure explainer
  - Release notes draft
- If the key is missing, these workflows no-op and do not block CI.

## Generated Repo Contents (Highlights)
- `docs/` with: QUALITY, DEPLOYMENT, RUNBOOK, RELEASE, ARCHITECTURE, SECURITY, ENV_VARS, METRICS, GITHUB_SETTINGS.
- `.github/workflows/`: CI, release, deploy staging/prod, AI automation.
- `.github/pull_request_template.md`, `.github/ISSUE_TEMPLATE/*`, `.github/dependabot.yml`.
- `compose/compose.staging.yml`, `compose/compose.prod.yml`.
- `Dockerfile`, `Makefile`, `.env.example`.
- Deploy logs: `docs/DEPLOY_LOG.md` and metrics JSONL: `docs/DEPLOY_METRICS.jsonl`.

## Post-init Checklist (New Repo)
1) Apply GitHub settings from `docs/GITHUB_SETTINGS.md`.
2) Add required secrets: `SSH_HOST`, `SSH_USER`, `SSH_KEY`, `SSH_PORT`, optional `OPENAI_API_KEY`.
3) Prepare servers and `.env` files for staging/production (see `docs/DEPLOYMENT.md`).
4) Implement a minimal `/health` endpoint and structured logging.
5) Configure `make migrate-check` when migrations are introduced.

## Agent Automation Endpoints
For chat/assistant orchestration without the UI:

- `POST /api/agent/proposal`
  - Headers: `Content-Type: application/json`, `X-Agent-Token: <AGENT_TOKEN>`
  - Body: `{ "plan": "...", "meta": { ... } }`

- `POST /api/agent/init`
  - Headers: `Content-Type: application/json`, `X-Agent-Token: <AGENT_TOKEN>`
  - Body: `{ "proposal_id": "...", "visibility": "private|public", "stack": "python|node", "tier": "1|2", "automation_level": "repo_only|repo_ci|repo_ci_cd", "deploy_mode": "none|ssh_compose", "meta": { ... } }`

## Data & Cleanup
- Generated proposals are saved in `out/`.
- Files are cleaned 30 minutes after a download.

## Security
- API keys remain server-side.
- For remote access, use Cloudflare Access and/or set `INIT_TOKEN`.
- Agent endpoints are disabled unless `AGENT_TOKEN` is configured.

## Project Structure
- `cmd/docgen/main.go` — server bootstrap
- `internal/` — httpapi, AI, scaffolding, project init, stack catalog, progress
- `templates/` — proposal, README, docs, workflows, compose, Makefile, stack config
- `public/` — UI assets
- `start-server` — build + systemd service setup
- `devops-standard-coverage.md` — coverage checklist vs standard

## License
Proprietary — All rights reserved. No use, copying, or distribution without permission.
