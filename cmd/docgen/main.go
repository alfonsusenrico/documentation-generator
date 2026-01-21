package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"proposal/internal/ai"
	"proposal/internal/config"
	"proposal/internal/httpapi"
	"proposal/internal/scaffold"
	"proposal/internal/stack"
	"proposal/internal/storage"
	"proposal/internal/templates"
)

func main() {
	cfg := config.Load()
	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		log.Fatal(err)
	}

	tmpl := templates.NewManager(cfg.TemplatesDir)
	stackCatalog, err := stack.LoadCatalog(filepath.Join(cfg.TemplatesDir, "stack.json"))
	if err != nil {
		log.Fatal(err)
	}

	aiClient := ai.NewClient(cfg.OpenAIKey, cfg.OpenAIModel, cfg.Lang)
	store := storage.NewStore(cfg.OutDir, cfg.CleanupDelay)
	scaffoldGen := &scaffold.Generator{
		Templates: tmpl,
		AI:        aiClient,
		Stacks:    stackCatalog,
	}

	server := httpapi.NewServer(cfg, tmpl, aiClient, store, scaffoldGen)

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("public")))
	server.RegisterRoutes(mux)

	log.Println("http://localhost:" + cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}
