#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: init-project.sh --client <client-slug> --project <project-slug> --repo <repo-name> \
  --visibility <public|private> --owner <owner> --readme <path> --gitignore <path>
EOF
}

client=""
project=""
repo=""
visibility=""
owner=""
readme_path=""
gitignore_path=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --client)
      client="${2:-}"; shift 2 ;;
    --project)
      project="${2:-}"; shift 2 ;;
    --repo)
      repo="${2:-}"; shift 2 ;;
    --visibility)
      visibility="${2:-}"; shift 2 ;;
    --owner)
      owner="${2:-}"; shift 2 ;;
    --readme)
      readme_path="${2:-}"; shift 2 ;;
    --gitignore)
      gitignore_path="${2:-}"; shift 2 ;;
    -h|--help)
      usage; exit 0 ;;
    *)
      echo "Unknown arg: $1" >&2
      usage
      exit 1 ;;
  esac
done

if [[ -z "$client" || -z "$project" || -z "$repo" || -z "$visibility" || -z "$owner" || -z "$readme_path" || -z "$gitignore_path" ]]; then
  usage
  exit 1
fi

if [[ "$visibility" != "public" && "$visibility" != "private" ]]; then
  echo "Visibility must be public or private." >&2
  exit 1
fi

if [[ ! -f "$readme_path" ]]; then
  echo "README template output not found: $readme_path" >&2
  exit 1
fi

if [[ ! -f "$gitignore_path" ]]; then
  echo "Gitignore template not found: $gitignore_path" >&2
  exit 1
fi

if ! command -v git >/dev/null 2>&1; then
  echo "git is required but not found in PATH." >&2
  exit 1
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is required but not found in PATH." >&2
  exit 1
fi

projects_root="${HOME}/project"
client_dir="${projects_root}/${client}"
project_dir="${client_dir}/${project}"

if [[ -e "$project_dir" ]]; then
  echo "Project directory already exists: $project_dir" >&2
  exit 1
fi

mkdir -p "$client_dir"
mkdir "$project_dir"

cp "$readme_path" "${project_dir}/README.md"
cp "$gitignore_path" "${project_dir}/.gitignore"

cd "$project_dir"
git init -b main

git_name="${GIT_USER_NAME:-}"
git_email="${GIT_USER_EMAIL:-}"
global_name="$(git config --global user.name || true)"
global_email="$(git config --global user.email || true)"
if [[ -z "$git_name" ]]; then
  git_name="$global_name"
fi
if [[ -z "$git_email" ]]; then
  git_email="$global_email"
fi
if [[ -z "$git_name" || -z "$git_email" ]]; then
  echo "Git identity missing. Set GIT_USER_NAME/GIT_USER_EMAIL in .env or run:" >&2
  echo "  git config --global user.name \"Your Name\"" >&2
  echo "  git config --global user.email \"you@example.com\"" >&2
  exit 1
fi

git config user.name "$git_name"
git config user.email "$git_email"

git add README.md .gitignore
git commit -m "chore: init project"

repo_full="${owner}/${repo}"
visibility_flag="--private"
if [[ "$visibility" == "public" ]]; then
  visibility_flag="--public"
fi

gh repo create "$repo_full" "$visibility_flag" --source . --remote origin --confirm
git remote set-url origin "https://github.com/${repo_full}.git"
gh auth setup-git >/dev/null 2>&1 || true
git push -u origin main

echo "Initialized ${project_dir}"
