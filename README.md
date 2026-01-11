# Documentation Generator

![Status](https://img.shields.io/badge/status-in--development-blue)

## Summary
Documentation Generator is a private, local-first tool that turns project notes into structured proposals and then initializes a ready-to-use GitHub repository with a project README. It is designed for a single developer workflow and runs on a home server.

## Core Capabilities
- Generate a proposal from a free-text plan using a fixed template (Markdown + optional PDF).
- Reuse past proposals without re-generating.
- Initiate a project: create `~/project/<client>/<project>`, initialize git, generate README, and push to GitHub via `gh`.
- Local storage for outputs with automatic cleanup after download.

## Workflow
1) Paste a project plan and generate a proposal.
2) Review the proposal in the UI.
3) Initiate the project to create the repo and README.

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
| GITHUB_OWNER | no | alfonsusenrico | GitHub owner/org slug |
| GIT_USER_NAME | no | Enrico | Optional git identity override |
| GIT_USER_EMAIL | no | you@example.com | Optional git identity override |

## Usage
### Generate a proposal
- Fill in project name, client/owner, and the plan.
- Click **Generate proposal** to produce Markdown (and PDF if enabled).

### Load existing proposals
- Use **Existing Proposals** to load from `out/` without re-generating.

### Initiate a project
- Select visibility (private/public) and click **Initiate project**.
- The system creates: `~/project/<client>/<project>` and pushes to GitHub.

## Data & Cleanup
- Generated proposals are saved in `out/`.
- Files are cleaned 30 minutes after a download.

## Security
- API keys remain server-side.
- For remote access, use Cloudflare Access and/or set `INIT_TOKEN`.

## Project Structure
- `main.go` — HTTP server and OpenAI integration
- `public/` — UI assets
- `PROPOSAL_TEMPLATE.md` — proposal template
- `README_TEMPLATE.md` — project README template
- `init-project.sh` — project initialization script
- `start-server` — build + systemd service setup

## License
Proprietary — All rights reserved. No use, copying, or distribution without permission.
