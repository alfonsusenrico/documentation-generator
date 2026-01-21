package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"proposal/internal/progress"
)

type InitRequest struct {
	Owner        string
	RepoName     string
	Visibility   string
	ClientSlug   string
	ProjectSlug  string
	GitUserName  string
	GitUserEmail string
	Files        []File
	Reporter     progress.Reporter
}

type File struct {
	Path    string
	Content string
}

type InitResult struct {
	ProjectPath string
	RepoURL     string
	RepoName    string
}

func Init(req InitRequest) (InitResult, error) {
	if req.Owner == "" || req.RepoName == "" {
		return InitResult{}, fmt.Errorf("missing repo owner or name")
	}
	if req.Visibility != "public" && req.Visibility != "private" {
		return InitResult{}, fmt.Errorf("visibility must be public or private")
	}
	if len(req.Files) == 0 {
		return InitResult{}, fmt.Errorf("no scaffold files to write")
	}
	if err := requireCommand("git"); err != nil {
		return InitResult{}, err
	}
	if err := requireCommand("gh"); err != nil {
		return InitResult{}, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return InitResult{}, fmt.Errorf("home directory not available")
	}
	env := []string{"HOME=" + homeDir}

	clientDir := filepath.Join(homeDir, "project", req.ClientSlug)
	projectDir := filepath.Join(clientDir, req.ProjectSlug)
	if _, err := os.Stat(projectDir); err == nil {
		return InitResult{}, fmt.Errorf("project directory already exists: %s", projectDir)
	}

	if err := os.MkdirAll(clientDir, 0o755); err != nil {
		return InitResult{}, fmt.Errorf("create client directory: %w", err)
	}
	if err := os.Mkdir(projectDir, 0o755); err != nil {
		return InitResult{}, fmt.Errorf("create project directory: %w", err)
	}

	report(req.Reporter, "Writing scaffold files", projectDir)
	for _, f := range req.Files {
		target := filepath.Join(projectDir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return InitResult{}, fmt.Errorf("create dir for %s: %w", f.Path, err)
		}
		if err := os.WriteFile(target, []byte(f.Content), 0o644); err != nil {
			return InitResult{}, fmt.Errorf("write %s: %w", f.Path, err)
		}
	}

	report(req.Reporter, "Initializing git", "Creating initial commit.")
	if err := run(projectDir, env, "git", "init", "-b", "main"); err != nil {
		return InitResult{}, err
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
		return InitResult{}, fmt.Errorf("git identity missing; set GIT_USER_NAME/GIT_USER_EMAIL or configure global git")
	}

	if err := run(projectDir, env, "git", "config", "user.name", gitName); err != nil {
		return InitResult{}, err
	}
	if err := run(projectDir, env, "git", "config", "user.email", gitEmail); err != nil {
		return InitResult{}, err
	}

	if err := run(projectDir, env, "git", "add", "."); err != nil {
		return InitResult{}, err
	}
	if err := run(projectDir, env, "git", "commit", "-m", "chore: init project [skip ci]"); err != nil {
		return InitResult{}, err
	}

	report(req.Reporter, "Publishing repository", "Creating remote and pushing.")
	repoFull := req.Owner + "/" + req.RepoName
	visibilityFlag := "--private"
	if req.Visibility == "public" {
		visibilityFlag = "--public"
	}
	if err := run(projectDir, env, "gh", "repo", "create", repoFull, visibilityFlag, "--source", ".", "--remote", "origin", "--confirm"); err != nil {
		return InitResult{}, err
	}
	if err := run(projectDir, env, "git", "remote", "set-url", "origin", "https://github.com/"+repoFull+".git"); err != nil {
		return InitResult{}, err
	}
	_ = run(projectDir, env, "gh", "auth", "setup-git")
	if err := run(projectDir, env, "git", "push", "-u", "origin", "main"); err != nil {
		return InitResult{}, err
	}
	if err := run(projectDir, env, "git", "branch", "dev"); err != nil {
		return InitResult{}, err
	}
	if err := run(projectDir, env, "git", "push", "-u", "origin", "dev"); err != nil {
		return InitResult{}, err
	}

	return InitResult{
		ProjectPath: projectDir,
		RepoURL:     "https://github.com/" + repoFull,
		RepoName:    req.RepoName,
	}, nil
}

func report(reporter progress.Reporter, message, detail string) {
	if reporter == nil {
		return
	}
	reporter.Report(progress.Event{Message: message, Detail: detail})
}

func requireCommand(cmd string) error {
	if _, err := exec.LookPath(cmd); err != nil {
		return fmt.Errorf("%s is required but not found in PATH", cmd)
	}
	return nil
}

func run(dir string, env []string, cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Dir = dir
	if len(env) > 0 {
		c.Env = append(os.Environ(), env...)
	}
	out, err := c.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s: %s", cmd, msg)
	}
	return nil
}

func gitConfig(key string) (string, error) {
	out, err := exec.Command("git", "config", "--global", key).CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
