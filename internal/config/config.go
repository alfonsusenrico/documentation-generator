package config

import (
	"os"
	"strings"
	"time"
)

const (
	DefaultOutDir      = "out"
	DefaultTemplates   = "templates"
	DefaultModel       = "gpt-5-mini"
	DefaultPort        = "3000"
	DefaultLang        = "en"
	DefaultCleanupDelay = 30 * time.Minute
)

type Config struct {
	OutDir        string
	TemplatesDir  string
	OpenAIKey     string
	OpenAIModel   string
	Port          string
	PublicBaseURL string
	Lang          string
	PreparedBy    string
	EnablePDF     bool
	InitToken     string
	GithubOwner   string
	GitUserName   string
	GitUserEmail  string
	CleanupDelay  time.Duration
}

func Load() Config {
	lang := strings.ToLower(strings.TrimSpace(os.Getenv("LANG")))
	if lang != "id" && lang != "en" {
		lang = DefaultLang
	}
	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if model == "" {
		model = DefaultModel
	}
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = DefaultPort
	}

	return Config{
		OutDir:        DefaultOutDir,
		TemplatesDir:  DefaultTemplates,
		OpenAIKey:     strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		OpenAIModel:   model,
		Port:          port,
		PublicBaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/"),
		Lang:          lang,
		PreparedBy:    strings.TrimSpace(os.Getenv("PREPARED_BY")),
		EnablePDF:     strings.TrimSpace(os.Getenv("ENABLE_PDF")) == "1",
		InitToken:     strings.TrimSpace(os.Getenv("INIT_TOKEN")),
		GithubOwner:   strings.TrimSpace(os.Getenv("GITHUB_OWNER")),
		GitUserName:   strings.TrimSpace(os.Getenv("GIT_USER_NAME")),
		GitUserEmail:  strings.TrimSpace(os.Getenv("GIT_USER_EMAIL")),
		CleanupDelay:  DefaultCleanupDelay,
	}
}
