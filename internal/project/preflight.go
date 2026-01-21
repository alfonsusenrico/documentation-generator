package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type PreflightRequest struct {
	Owner        string
	RepoName     string
	Visibility   string
	ClientSlug   string
	ProjectSlug  string
	GitUserName  string
	GitUserEmail string
}

type PreflightResult struct {
	ProjectPath string
	RepoFull    string
}

func Preflight(req PreflightRequest) (PreflightResult, error) {
	if req.Owner == "" || req.RepoName == "" {
		return PreflightResult{}, fmt.Errorf("missing repo owner or name")
	}
	if req.Visibility != "public" && req.Visibility != "private" {
		return PreflightResult{}, fmt.Errorf("visibility must be public or private")
	}
	if req.ClientSlug == "" || req.ProjectSlug == "" {
		return PreflightResult{}, fmt.Errorf("missing client or project slug")
	}
	if err := requireCommand("git"); err != nil {
		return PreflightResult{}, err
	}
	if err := requireCommand("gh"); err != nil {
		return PreflightResult{}, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return PreflightResult{}, fmt.Errorf("home directory not available")
	}
	env := []string{"HOME=" + homeDir}

	projectDir := filepath.Join(homeDir, "project", req.ClientSlug, req.ProjectSlug)
	if info, err := os.Stat(projectDir); err == nil {
		if info.IsDir() {
			return PreflightResult{}, fmt.Errorf("project directory already exists: %s", projectDir)
		}
		return PreflightResult{}, fmt.Errorf("project path exists and is not a directory: %s", projectDir)
	} else if !os.IsNotExist(err) {
		return PreflightResult{}, fmt.Errorf("check project directory: %w", err)
	}

	gitName := strings.TrimSpace(req.GitUserName)
	gitEmail := strings.TrimSpace(req.GitUserEmail)
	if gitName == "" {
		gitName, _ = gitConfig("user.name")
	}
	if gitEmail == "" {
		gitEmail, _ = gitConfig("user.email")
	}
	if gitName == "" || gitEmail == "" {
		return PreflightResult{}, fmt.Errorf("git identity missing; set GIT_USER_NAME/GIT_USER_EMAIL or configure global git")
	}

	if err := run("", env, "gh", "auth", "status"); err != nil {
		return PreflightResult{}, fmt.Errorf("gh auth status: %w", err)
	}

	repoFull := req.Owner + "/" + req.RepoName
	exists, err := repoExists(env, repoFull)
	if err != nil {
		return PreflightResult{}, err
	}
	if exists {
		return PreflightResult{}, fmt.Errorf("repo already exists: %s", repoFull)
	}

	return PreflightResult{
		ProjectPath: projectDir,
		RepoFull:    repoFull,
	}, nil
}

func repoExists(env []string, repoFull string) (bool, error) {
	c := exec.Command("gh", "repo", "view", repoFull)
	if len(env) > 0 {
		c.Env = append(os.Environ(), env...)
	}
	out, err := c.CombinedOutput()
	if err == nil {
		return true, nil
	}
	msg := strings.ToLower(strings.TrimSpace(string(out)))
	if msg == "" {
		return false, nil
	}
	if strings.Contains(msg, "not found") || strings.Contains(msg, "could not resolve") || strings.Contains(msg, "http 404") {
		return false, nil
	}
	return false, fmt.Errorf("gh repo view failed: %s", strings.TrimSpace(string(out)))
}
