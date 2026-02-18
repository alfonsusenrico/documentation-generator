package scaffold

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"proposal/internal/ai"
	"proposal/internal/progress"
	"proposal/internal/stack"
	"proposal/internal/storage"
	"proposal/internal/templates"
	"proposal/internal/util"
)

type Generator struct {
	Templates *templates.Manager
	AI        *ai.Client
	Stacks    stack.Catalog
}

type Request struct {
	Proposal        string
	Meta            map[string]interface{}
	Stack           string
	Tier            string
	Visibility      string
	AutomationLevel string
	DeployMode      string
	GithubOwner     string
	PreparedBy      string
	Reporter        progress.Reporter
}

type File struct {
	Path    string
	Content string
}

type Result struct {
	Files       []File
	RepoName    string
	RepoURL     string
	ClientSlug  string
	ProjectSlug string
	ProjectName string
	ClientName  string
	ImageName   string
	AppPort     int
}

var docFiles = []string{
	"QUALITY.md",
	"DEPLOYMENT.md",
	"RUNBOOK.md",
	"RELEASE.md",
	"ARCHITECTURE.md",
	"SECURITY.md",
	"ENV_VARS.md",
	"METRICS.md",
}

func DocFileCount() int {
	return len(docFiles)
}

func (g *Generator) Generate(req Request) (Result, error) {
	if strings.TrimSpace(req.Proposal) == "" {
		return Result{}, fmt.Errorf("proposal content is empty")
	}
	if req.Stack == "" {
		return Result{}, fmt.Errorf("stack is required")
	}
	if req.Tier != "1" && req.Tier != "2" {
		return Result{}, fmt.Errorf("tier must be 1 or 2")
	}
	if req.AutomationLevel == "" {
		req.AutomationLevel = "repo_ci"
	}
	if req.AutomationLevel != "repo_only" && req.AutomationLevel != "repo_ci" && req.AutomationLevel != "repo_ci_cd" {
		return Result{}, fmt.Errorf("automation_level must be repo_only, repo_ci, or repo_ci_cd")
	}
	if req.DeployMode == "" {
		req.DeployMode = "none"
	}
	if req.DeployMode != "none" && req.DeployMode != "ssh_compose" {
		return Result{}, fmt.Errorf("deploy_mode must be none or ssh_compose")
	}
	stackCfg, err := g.Stacks.Get(req.Stack)
	if err != nil {
		return Result{}, err
	}

	meta := map[string]interface{}{}
	for k, v := range req.Meta {
		meta[k] = v
	}

	clientName := storage.EnsureMetaString(meta, "client_or_owner", "Project Client")
	projectName := storage.EnsureMetaString(meta, "project_name", "Project")
	if req.PreparedBy != "" {
		storage.EnsureMetaString(meta, "your_name", req.PreparedBy)
	}

	owner := strings.TrimSpace(req.GithubOwner)
	if owner == "" {
		owner = "alfonsusenrico"
	}

	clientSlug := util.Slugify(clientName, "client")
	projectSlug := util.Slugify(projectName, "project")
	repoName := clientSlug + "-" + projectSlug
	repoURL := "https://github.com/" + owner + "/" + repoName
	imageName := "ghcr.io/" + owner + "/" + repoName

	meta["status_slug"] = "pre--development"
	meta["status_color"] = "blue"
	meta["clone_command"] = "git clone " + repoURL + "\ncd " + repoName
	if req.Visibility == "public" {
		meta["license"] = "MIT License - permissive; see https://opensource.org/license/mit/"
	} else {
		meta["license"] = "Proprietary - All rights reserved. No use, copying, or distribution without permission."
	}
	meta["repo_url"] = repoURL
	meta["service_name"] = repoName
	meta["image_name"] = imageName
	meta["app_port"] = stackCfg.AppPort
	meta["health_endpoint"] = "/health"
	meta["stack"] = req.Stack
	meta["tier"] = req.Tier
	meta["automation_level"] = req.AutomationLevel
	meta["deploy_mode"] = req.DeployMode

	values := buildValues(projectName, clientName, repoName, repoURL, imageName, stackCfg)
	values["AUTOMATION_LEVEL"] = req.AutomationLevel
	values["DEPLOY_MODE"] = req.DeployMode
	values["STATUS_SLUG"] = "pre--development"
	values["STATUS_COLOR"] = "blue"
	values["CLONE_COMMAND"] = "git clone " + repoURL + "\ncd " + repoName
	if license, ok := meta["license"].(string); ok {
		values["LICENSE"] = license
	}

	files := []File{}

	// Deterministic templates
	files, err = g.addDeterministic(files, "Dockerfile", "Dockerfile", values)
	if err != nil {
		return Result{}, err
	}
	files, err = g.addDeterministic(files, "Makefile", "Makefile", values)
	if err != nil {
		return Result{}, err
	}
	files, err = g.addDeterministic(files, "env/.env.example", ".env.example", values)
	if err != nil {
		return Result{}, err
	}
	files, err = g.addDeterministic(files, "gitignore/.gitignore", ".gitignore", values)
	if err != nil {
		return Result{}, err
	}
	files, err = g.addDeterministic(files, "compose/compose.staging.yml", "compose/compose.staging.yml", values)
	if err != nil {
		return Result{}, err
	}
	files, err = g.addDeterministic(files, "compose/compose.prod.yml", "compose/compose.prod.yml", values)
	if err != nil {
		return Result{}, err
	}

	files, err = g.addDeterministic(files, "github/pull_request_template.md", ".github/pull_request_template.md", values)
	if err != nil {
		return Result{}, err
	}
	files, err = g.addDeterministic(files, "github/ISSUE_TEMPLATE/bug_report.md", ".github/ISSUE_TEMPLATE/bug_report.md", values)
	if err != nil {
		return Result{}, err
	}
	files, err = g.addDeterministic(files, "github/ISSUE_TEMPLATE/feature_request.md", ".github/ISSUE_TEMPLATE/feature_request.md", values)
	if err != nil {
		return Result{}, err
	}

	files, err = g.addDeterministic(files, "docs/GITHUB_SETTINGS.md", "docs/GITHUB_SETTINGS.md", values)
	if err != nil {
		return Result{}, err
	}
	files, err = g.addDeterministic(files, "automation/automation.contract.yaml", "automation.contract.yaml", values)
	if err != nil {
		return Result{}, err
	}
	files, err = g.addDeterministic(files, "automation/AUTOMATION.md", "docs/AUTOMATION.md", values)
	if err != nil {
		return Result{}, err
	}

	if req.AutomationLevel != "repo_only" {
		dependabotPath := filepath.Join("github/dependabot", req.Stack+".yml")
		files, err = g.addDeterministic(files, dependabotPath, ".github/dependabot.yml", values)
		if err != nil {
			return Result{}, err
		}

		ciPath := filepath.Join("github/workflows", req.Stack, "tier"+req.Tier, "ci.yml")
		files, err = g.addDeterministic(files, ciPath, ".github/workflows/ci.yml", values)
		if err != nil {
			return Result{}, err
		}

		aiWorkflowFiles := []string{"ai-pr-summary.yml", "ai-ci-explainer.yml"}
		for _, wf := range aiWorkflowFiles {
			path := filepath.Join("github/workflows", wf)
			files, err = g.addDeterministic(files, path, filepath.Join(".github/workflows", wf), values)
			if err != nil {
				return Result{}, err
			}
		}
	}

	if req.AutomationLevel == "repo_ci_cd" && req.DeployMode == "ssh_compose" {
		workflowFiles := []string{"release.yml", "deploy-staging.yml", "deploy-production.yml"}
		for _, wf := range workflowFiles {
			path := filepath.Join("github/workflows", req.Stack, wf)
			files, err = g.addDeterministic(files, path, filepath.Join(".github/workflows", wf), values)
			if err != nil {
				return Result{}, err
			}
		}
	}

	// AI-filled templates
	docsDir := filepath.Join("docs", "tier"+req.Tier)
	totalDocs := len(docFiles) + 1
	for i, doc := range docFiles {
		report(req.Reporter, fmt.Sprintf("Generating docs %d/%d", i+1, totalDocs), doc)
		path := filepath.Join(docsDir, doc)
		content, err := g.generateAI(path, values, meta, req.Proposal)
		if err != nil {
			return Result{}, err
		}
		files = append(files, File{Path: filepath.Join("docs", doc), Content: content})
	}

	report(req.Reporter, fmt.Sprintf("Generating docs %d/%d", totalDocs, totalDocs), "README.md")
	readmeContent, err := g.generateReadme(values, meta, req.Proposal)
	if err != nil {
		return Result{}, err
	}
	files = append(files, File{Path: "README.md", Content: readmeContent})

	return Result{
		Files:       files,
		RepoName:    repoName,
		RepoURL:     repoURL,
		ClientSlug:  clientSlug,
		ProjectSlug: projectSlug,
		ProjectName: projectName,
		ClientName:  clientName,
		ImageName:   imageName,
		AppPort:     stackCfg.AppPort,
	}, nil
}

func (g *Generator) addDeterministic(files []File, templatePath, targetPath string, values map[string]string) ([]File, error) {
	content, err := g.Templates.Read(templatePath)
	if err != nil {
		return nil, err
	}
	content = templates.Apply(content, values)
	if err := templates.EnsureNoPlaceholders(content); err != nil {
		return nil, fmt.Errorf("%s: %w", templatePath, err)
	}
	return append(files, File{Path: targetPath, Content: content}), nil
}

func report(reporter progress.Reporter, message, detail string) {
	if reporter == nil {
		return
	}
	reporter.Report(progress.Event{Message: message, Detail: detail})
}

func (g *Generator) generateAI(templatePath string, values map[string]string, meta map[string]interface{}, proposal string) (string, error) {
	tpl, err := g.Templates.Read(templatePath)
	if err != nil {
		return "", err
	}
	prepared := templates.Apply(tpl, values)
	out, err := g.AI.GenerateDoc(prepared, meta, proposal)
	if err != nil {
		return "", err
	}
	if err := templates.EnsureNoPlaceholders(out); err != nil {
		return "", err
	}
	return out, nil
}

func (g *Generator) generateReadme(values map[string]string, meta map[string]interface{}, proposal string) (string, error) {
	tpl, err := g.Templates.Read(filepath.Join("readme", "README_TEMPLATE.md"))
	if err != nil {
		return "", err
	}
	prepared := templates.Apply(tpl, values)
	out, err := g.AI.GenerateReadme(prepared, meta, proposal)
	if err != nil {
		return "", err
	}
	if err := templates.EnsureNoPlaceholders(out); err != nil {
		return "", err
	}
	return out, nil
}

func buildValues(projectName, clientName, repoName, repoURL, imageName string, stackCfg stack.Config) map[string]string {
	appPort := strconv.Itoa(stackCfg.AppPort)
	projectSlug := util.Slugify(projectName, "project")
	values := map[string]string{
		"PROJECT_NAME":         projectName,
		"CLIENT_OR_OWNER":      clientName,
		"SERVICE_NAME":         repoName,
		"IMAGE_NAME":           imageName,
		"REPO_URL":             repoURL,
		"APP_PORT":             appPort,
		"HEALTH_ENDPOINT":      "/health",
		"DEPLOY_PATH_STAGING":  "/opt/" + projectSlug + "/staging",
		"DEPLOY_PATH_PROD":     "/opt/" + projectSlug + "/production",
		"DEPLOY_LOG_PATH":      "docs/DEPLOY_LOG.md",
		"METRICS_LOG_PATH":     "docs/DEPLOY_METRICS.jsonl",
		"INSTALL_CMD":          stackCfg.Commands.Install,
		"LINT_CMD":             stackCfg.Commands.Lint,
		"TYPE_CMD":             stackCfg.Commands.Type,
		"TEST_CMD":             stackCfg.Commands.Test,
		"INTEGRATION_TEST_CMD": stackCfg.Commands.TestIntegration,
		"MIGRATE_CHECK_CMD":    stackCfg.Commands.MigrateCheck,
		"BUILD_CMD":            stackCfg.Commands.Build,
		"DEP_AUDIT_CMD":        stackCfg.Commands.Scan,
		"DOCKER_BASE_IMAGE":    stackCfg.Docker.BaseImage,
		"DOCKER_COPY_DEPS":     stackCfg.Docker.CopyDeps,
		"DOCKER_INSTALL_STEP":  stackCfg.Docker.InstallStep,
		"DOCKER_BUILD_STEP":    stackCfg.Docker.BuildStep,
		"DOCKER_START_CMD":     stackCfg.Docker.StartCmd,
	}
	return values
}
