package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

const (
	outDir                = "out"
	proposalTemplatePath  = "PROPOSAL_TEMPLATE.md"
	readmeTemplatePath    = "README_TEMPLATE.md"
	gitignoreTemplatePath = "GITIGNORE_DEFAULT"
	initScriptPath        = "init-project.sh"
	cleanupDelay          = 30 * time.Minute
)

var (
	cleanupMu     sync.Mutex
	cleanupTimers = map[string]*time.Timer{}
)

type proposalReq struct {
	Plan string                 `json:"plan"`
	Meta map[string]interface{} `json:"meta"`
}

type proposalResp struct {
	ID        string `json:"id"`
	Markdown  string `json:"markdown"`
	PDFReady  bool   `json:"pdf_ready"`
	Downloads struct {
		MD  string `json:"md"`
		PDF string `json:"pdf"`
	} `json:"downloads"`
}

type proposalSummary struct {
	ID          string    `json:"id"`
	Label       string    `json:"label"`
	ProjectName string    `json:"project_name"`
	ClientOwner string    `json:"client_or_owner"`
	Date        string    `json:"date"`
	Version     string    `json:"version"`
	PDFReady    bool      `json:"pdf_ready"`
	ModTime     time.Time `json:"-"`
}

type proposalDetailResp struct {
	ID        string            `json:"id"`
	Markdown  string            `json:"markdown"`
	PDFReady  bool              `json:"pdf_ready"`
	Meta      map[string]string `json:"meta"`
	Downloads struct {
		MD  string `json:"md"`
		PDF string `json:"pdf"`
	} `json:"downloads"`
}

type initReq struct {
	ProposalID string                 `json:"proposal_id"`
	Visibility string                 `json:"visibility"`
	Meta       map[string]interface{} `json:"meta"`
}

type initResp struct {
	ProjectPath  string   `json:"project_path"`
	RepoURL      string   `json:"repo_url"`
	RepoName     string   `json:"repo_name"`
	NextCommands []string `json:"next_commands"`
}

type proposalMeta struct {
	ProjectName string
	ClientOwner string
	PreparedBy  string
	Date        string
	Version     string
}

func main() {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatal(err)
	}

	http.Handle("/", http.FileServer(http.Dir("public")))
	http.HandleFunc("/api/proposal", handleProposal)
	http.HandleFunc("/api/init", handleInit)
	http.HandleFunc("/api/proposals", handleProposals)
	http.HandleFunc("/api/proposals/", handleProposalDetail)
	http.HandleFunc("/download/", handleDownload)

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "3000"
	}

	log.Println("http://localhost:" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleProposal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req proposalReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Plan = strings.TrimSpace(req.Plan)
	if req.Plan == "" {
		writeJSONError(w, http.StatusBadRequest, "missing plan")
		return
	}
	if req.Meta == nil {
		req.Meta = map[string]interface{}{}
	}

	tpl, err := os.ReadFile(proposalTemplatePath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "missing PROPOSAL_TEMPLATE.md")
		return
	}

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		writeJSONError(w, http.StatusInternalServerError, "OPENAI_API_KEY not set")
		return
	}

	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if model == "" {
		model = "gpt-5-mini"
	}

	lang := strings.ToLower(strings.TrimSpace(os.Getenv("LANG")))
	if lang != "id" && lang != "en" {
		lang = "en"
	}

	preparedBy := strings.TrimSpace(os.Getenv("PREPARED_BY"))
	enablePDF := strings.TrimSpace(os.Getenv("ENABLE_PDF")) == "1"
	publicBaseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")

	// Defaults (provided facts, not invented)
	loc, _ := time.LoadLocation("Asia/Jakarta")
	today := time.Now().In(loc).Format("2006-01-02")

	if v, ok := req.Meta["date"].(string); !ok || strings.TrimSpace(v) == "" {
		req.Meta["date"] = today
	}
	if v, ok := req.Meta["version"].(string); !ok || strings.TrimSpace(v) == "" {
		req.Meta["version"] = "v0.1"
	}
	if v, ok := req.Meta["client_or_owner"].(string); !ok || strings.TrimSpace(v) == "" {
		req.Meta["client_or_owner"] = "Project Client"
	}
	if preparedBy != "" {
		if v, ok := req.Meta["your_name"].(string); !ok || strings.TrimSpace(v) == "" {
			req.Meta["your_name"] = preparedBy
		}
	}

	langRule := "Write the proposal body content in English."
	if lang == "id" {
		langRule = "Write the proposal body content in Indonesian."
	}

	instructions := strings.Join([]string{
		"You generate a client-readable project proposal in Markdown.",
		"Follow the provided template EXACTLY (same headings/order AND same Markdown formatting). Do not add, remove, rename, or reorder sections.",
		"Formatting rule: Match the template formatting exactly (same bold labels like **Client/Owner:**, same bullet list style, and the same table format). Do not convert required bullet lists into paragraphs.",
		"Keep template labels/headings in English exactly as provided; only the body content follows the requested language.",
		langRule,

		"Tone rule: Use professional neutral voice. Avoid first-person pronouns (I/we/our/me/my) and Indonesian equivalents (saya/kami/kita/aku). Write in third-person as a project document.",

		"Header rule: The header block (Project name, Client/Owner, Prepared by, Date, Version) MUST use META values when present. Do not output 'TBD' if META has a non-empty value.",
		"Replacement rule: Use META + PROJECT PLAN to fill every placeholder. If information is missing, write 'TBD' (never guess).",
		"Defaults rule: Date must use meta.date; Version must use meta.version; Client/Owner must use meta.client_or_owner; Prepared by must use meta.your_name. These are provided values, not guesses.",

		"Truthfulness rule: Do not invent calendar dates, prices, integrations, performance claims, or dependencies not present in META/PLAN. Relative timeline labels (Day 1–3, Week 1–2) are allowed.",
		"Timeline rule: Prefer relative schedule labels explicitly mentioned in the plan (e.g., Day 1/Day 2/Week 1). If the plan implies phases but lacks dates, fill Target date with reasonable relative durations (Day 1–3 or Week 1–3) instead of 'TBD'. Only use calendar dates if they appear in META/PLAN.",

		"Deployment reality rule: If mentioning PDF conversion, state that pandoc runs on the same host when ENABLE_PDF=1. Do not mention a separate container or host-level installation unless explicitly stated in META/PLAN.",

		"Ownership rule: If 'Ownership' is not specified in META/PLAN, default Source code ownership to meta.your_name for personal/internal projects; otherwise use meta.client_or_owner.",

		"Length rule: aim for ~1 page; hard maximum 2 pages. If content grows, compress wording and reduce bullet counts while keeping the template structure and required sections.",
		"Hard length rule: The proposal MUST fit within 2 PDF pages on A4 at 11pt, margin 18mm, line-spacing 1.5. If needed, shorten paragraphs and keep lists to max 3 bullets per section.",
		"Output ONLY the completed proposal in Markdown (no code fences, no commentary).",
	}, "\n")

	metaJSON, _ := json.MarshalIndent(req.Meta, "", "  ")
	input := "TEMPLATE:\n" + string(tpl) + "\n\nMETA:\n" + string(metaJSON) + "\n\nPROJECT PLAN:\n" + req.Plan

	client := openai.NewClient(option.WithAPIKey(apiKey))
	resp, err := client.Responses.New(context.TODO(), responses.ResponseNewParams{
		Model:        model,
		Instructions: openai.String(instructions),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(input),
		},
	})
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "openai error: "+err.Error())
		return
	}

	md := strings.TrimSpace(resp.OutputText())
	if md == "" {
		writeJSONError(w, http.StatusBadGateway, "empty model output")
		return
	}

	projectName, _ := req.Meta["project_name"].(string)
	clientOwner, _ := req.Meta["client_or_owner"].(string)
	version, _ := req.Meta["version"].(string)

	baseName := "Proposal_" + sanitizeName(projectName) + "-" + sanitizeName(clientOwner) + "_" + sanitizeName(version)
	mdDownloadName := baseName + ".md"
	pdfDownloadName := baseName + ".pdf"

	// Storage filenames remain ID-based (prevents collisions)
	id := randID(8)
	mdPath := filepath.Join(outDir, id+".md")
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "write md failed")
		return
	}

	pdfReady := false
	pdfPath := filepath.Join(outDir, id+".pdf")

	if enablePDF {
		if _, err := exec.LookPath("pandoc"); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "ENABLE_PDF=1 but pandoc not found in PATH")
			return
		}

		log.Printf("pandoc: converting %s -> %s", mdPath, pdfPath)
		out, err := exec.Command(
			"pandoc", mdPath, "-o", pdfPath,

			// Better font handling
			"--pdf-engine=xelatex",

			// Make it look like a modern doc
			"-V", "papersize=a4",
			"-V", "geometry:margin=18mm",
			"-V", "fontsize=11pt",
			"-V", "linestretch=1.5",
			"-V", "mainfont=DejaVu Serif",

			// IMPORTANT: remove pandoc "maketitle" behavior (huge top whitespace)
			"--metadata", "title=",
		).CombinedOutput()

		if err != nil {
			log.Printf("pandoc: FAILED: %v; output=%s", err, strings.TrimSpace(string(out)))
		} else {
			pdfReady = true
			if st, err := os.Stat(pdfPath); err == nil {
				log.Printf("pandoc: OK (%d bytes)", st.Size())
			} else {
				log.Printf("pandoc: OK (stat failed: %v)", err)
			}
		}
	}

	mdURL := "/download/" + id + ".md" + "?name=" + url.QueryEscape(mdDownloadName)
	pdfURL := "/download/" + id + ".pdf" + "?name=" + url.QueryEscape(pdfDownloadName)
	if publicBaseURL != "" {
		mdURL = publicBaseURL + mdURL
		pdfURL = publicBaseURL + pdfURL
	}

	var out proposalResp
	out.ID = id
	out.Markdown = md
	out.PDFReady = pdfReady
	out.Downloads.MD = mdURL
	out.Downloads.PDF = pdfURL

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func handleInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	initToken := strings.TrimSpace(os.Getenv("INIT_TOKEN"))
	if initToken != "" && r.Header.Get("X-Init-Token") != initToken {
		writeJSONError(w, http.StatusUnauthorized, "invalid init token")
		return
	}

	var req initReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.ProposalID = strings.TrimSpace(req.ProposalID)
	if req.ProposalID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing proposal_id")
		return
	}

	visibility := strings.ToLower(strings.TrimSpace(req.Visibility))
	if visibility != "public" && visibility != "private" {
		writeJSONError(w, http.StatusBadRequest, "visibility must be public or private")
		return
	}

	if req.Meta == nil {
		req.Meta = map[string]interface{}{}
	}

	clientName := ensureMetaString(req.Meta, "client_or_owner", "Project Client")
	projectName := ensureMetaString(req.Meta, "project_name", "Project")

	owner := strings.TrimSpace(os.Getenv("GITHUB_OWNER"))
	if owner == "" {
		owner = "alfonsusenrico"
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "home directory not available")
		return
	}

	clientSlug := slugify(clientName, "client")
	projectSlug := slugify(projectName, "project")
	repoName := clientSlug + "-" + projectSlug
	repoURL := "https://github.com/" + owner + "/" + repoName

	req.Meta["status_slug"] = "pre--development"
	req.Meta["status_color"] = "blue"
	req.Meta["clone_command"] = "git clone " + repoURL + "\ncd " + repoName
	if visibility == "public" {
		req.Meta["license"] = "MIT License — permissive; see https://opensource.org/license/mit/"
	} else {
		req.Meta["license"] = "Proprietary — All rights reserved. No use, copying, or distribution without permission."
	}

	proposalPath := filepath.Join(outDir, req.ProposalID+".md")
	proposalMD, err := os.ReadFile(proposalPath)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "proposal not found")
		return
	}

	readme, err := generateReadme(string(proposalMD), req.Meta)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "init output directory failed")
		return
	}

	tmpReadme, err := os.CreateTemp(outDir, req.ProposalID+"-readme-*.md")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "init temp readme failed")
		return
	}

	if _, err := tmpReadme.WriteString(readme); err != nil {
		_ = tmpReadme.Close()
		writeJSONError(w, http.StatusInternalServerError, "write readme failed")
		return
	}
	if err := tmpReadme.Close(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "close readme failed")
		return
	}
	defer os.Remove(tmpReadme.Name())

	baseDir, err := os.Getwd()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "working directory not available")
		return
	}

	scriptPath := filepath.Join(baseDir, initScriptPath)
	gitignorePath := filepath.Join(baseDir, gitignoreTemplatePath)
	if _, err := os.Stat(scriptPath); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "init script not found")
		return
	}
	if _, err := os.Stat(gitignorePath); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "gitignore template not found")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		scriptPath,
		"--client", clientSlug,
		"--project", projectSlug,
		"--repo", repoName,
		"--visibility", visibility,
		"--owner", owner,
		"--readme", tmpReadme.Name(),
		"--gitignore", gitignorePath,
	)
	cmd.Dir = baseDir
	cmd.Env = append(os.Environ(), "HOME="+homeDir)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		writeJSONError(w, http.StatusGatewayTimeout, "init timed out")
		return
	}
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		writeJSONError(w, http.StatusBadGateway, "init failed: "+msg)
		return
	}

	projectPath := filepath.Join(homeDir, "project", clientSlug, projectSlug)
	outResp := initResp{
		ProjectPath: projectPath,
		RepoURL:     repoURL,
		RepoName:    repoName,
		NextCommands: []string{
			"git clone " + repoURL,
			"cd " + repoName,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(outResp)
}

func handleProposals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string][]proposalSummary{"items": []proposalSummary{}})
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "read out directory failed")
		return
	}

	var list []proposalSummary
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		id := strings.TrimSuffix(name, ".md")
		if !isHexID(id) {
			continue
		}
		path := filepath.Join(outDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		meta := parseProposalMeta(string(content))
		pdfReady := fileExists(filepath.Join(outDir, id+".pdf"))
		label := buildProposalLabel(meta, id)
		modTime := time.Time{}
		if info, err := entry.Info(); err == nil {
			modTime = info.ModTime()
		}

		list = append(list, proposalSummary{
			ID:          id,
			Label:       label,
			ProjectName: meta.ProjectName,
			ClientOwner: meta.ClientOwner,
			Date:        meta.Date,
			Version:     meta.Version,
			PDFReady:    pdfReady,
			ModTime:     modTime,
		})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].ModTime.After(list[j].ModTime)
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string][]proposalSummary{"items": list})
}

func handleProposalDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/proposals/")
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "..") || !isHexID(id) {
		writeJSONError(w, http.StatusBadRequest, "invalid proposal id")
		return
	}

	path := filepath.Join(outDir, id+".md")
	content, err := os.ReadFile(path)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "proposal not found")
		return
	}

	meta := parseProposalMeta(string(content))
	pdfReady := fileExists(filepath.Join(outDir, id+".pdf"))

	baseName := "Proposal_" + sanitizeName(meta.ProjectName) + "-" + sanitizeName(meta.ClientOwner) + "_" + sanitizeName(meta.Version)
	mdName := baseName + ".md"
	pdfName := baseName + ".pdf"

	publicBaseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	mdURL := "/download/" + id + ".md" + "?name=" + url.QueryEscape(mdName)
	pdfURL := "/download/" + id + ".pdf" + "?name=" + url.QueryEscape(pdfName)
	if publicBaseURL != "" {
		mdURL = publicBaseURL + mdURL
		pdfURL = publicBaseURL + pdfURL
	}

	var out proposalDetailResp
	out.ID = id
	out.Markdown = strings.TrimSpace(string(content))
	out.PDFReady = pdfReady
	out.Meta = map[string]string{
		"project_name":    meta.ProjectName,
		"client_or_owner": meta.ClientOwner,
		"date":            meta.Date,
		"version":         meta.Version,
	}
	out.Downloads.MD = mdURL
	out.Downloads.PDF = pdfURL

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/download/")
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	path := filepath.Join(outDir, name)
	b, err := os.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	downloadName := r.URL.Query().Get("name")
	if downloadName == "" {
		downloadName = name
	}
	downloadName = filepath.Base(downloadName)
	downloadName = strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(downloadName)
	if downloadName == "" {
		downloadName = name
	}

	if strings.HasSuffix(name, ".pdf") {
		w.Header().Set("Content-Type", "application/pdf")
	} else {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	}

	w.Header().Set("Content-Disposition", `attachment; filename="`+downloadName+`"`)
	if _, err := w.Write(b); err != nil {
		log.Printf("download write failed: %v", err)
		return
	}

	if id := downloadID(name); id != "" {
		scheduleCleanup(id)
	}
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "TBD"
	}
	s = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '-', r == '_', r == '.':
			return r
		case r == ' ':
			return '_'
		default:
			return -1
		}
	}, s)
	s = strings.Trim(s, "_-.")
	if s == "" {
		return "TBD"
	}
	rs := []rune(s)
	if len(rs) > 80 {
		s = string(rs[:80])
	}
	return s
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isHexID(id string) bool {
	if len(id) != 16 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= '0' && r <= '9':
			continue
		case r >= 'a' && r <= 'f':
			continue
		default:
			return false
		}
	}
	return true
}

func parseProposalMeta(md string) proposalMeta {
	lines := strings.Split(md, "\n")
	meta := proposalMeta{}
	if len(lines) > 0 {
		title := strings.TrimSpace(strings.TrimSuffix(lines[0], "\r"))
		if strings.HasPrefix(title, "# Project Proposal") {
			rest := strings.TrimSpace(strings.TrimPrefix(title, "# Project Proposal"))
			rest = strings.TrimLeft(rest, "—-– ")
			meta.ProjectName = strings.TrimSpace(rest)
		}
	}

	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" {
			continue
		}
		if v := parseMetaLine(line, "Client/Owner"); v != "" {
			meta.ClientOwner = v
			continue
		}
		if v := parseMetaLine(line, "Prepared by"); v != "" {
			meta.PreparedBy = v
			continue
		}
		if v := parseMetaLine(line, "Date"); v != "" {
			meta.Date = v
			continue
		}
		if v := parseMetaLine(line, "Version"); v != "" {
			meta.Version = v
			continue
		}
	}

	return meta
}

func parseMetaLine(line, label string) string {
	prefix := "**" + label + ":**"
	if strings.HasPrefix(line, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(line, prefix))
	}
	return ""
}

func buildProposalLabel(meta proposalMeta, fallback string) string {
	label := strings.TrimSpace(meta.ProjectName)
	if label == "" {
		label = fallback
	}
	if meta.ClientOwner != "" {
		label += " - " + meta.ClientOwner
	}
	if meta.Date != "" {
		label += " (" + meta.Date + ")"
	}
	return label
}

func ensureMetaString(meta map[string]interface{}, key, fallback string) string {
	if v, ok := meta[key].(string); ok {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			meta[key] = trimmed
			return trimmed
		}
	}
	meta[key] = fallback
	return fallback
}

func slugify(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	needsDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if needsDash && b.Len() > 0 {
				b.WriteRune('-')
			}
			b.WriteRune(unicode.ToLower(r))
			needsDash = false
			continue
		}
		switch r {
		case ' ', '-', '_':
			needsDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return fallback
	}
	return out
}

func generateReadme(proposal string, meta map[string]interface{}) (string, error) {
	tpl, err := os.ReadFile(readmeTemplatePath)
	if err != nil {
		return "", fmt.Errorf("missing README_TEMPLATE.md")
	}

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}

	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if model == "" {
		model = "gpt-5-mini"
	}

	lang := strings.ToLower(strings.TrimSpace(os.Getenv("LANG")))
	if lang != "id" && lang != "en" {
		lang = "en"
	}

	langRule := "Write the README body content in English."
	if lang == "id" {
		langRule = "Write the README body content in Indonesian."
	}

	instructions := strings.Join([]string{
		"You generate a project README in Markdown.",
		"Follow the provided template EXACTLY (same headings/order AND same Markdown formatting). Do not add, remove, rename, or reorder sections.",
		"Formatting rule: Match the template formatting exactly (same bold labels, bullet list style, tables, and code fences). Do not convert required lists into paragraphs.",
		"Keep template labels/headings in English exactly as provided; only the body content follows the requested language.",
		langRule,

		"Tone rule: Use professional neutral voice. Avoid first-person pronouns (I/we/our/me/my) and Indonesian equivalents (saya/kami/kita/aku). Write in third-person as a project document.",
		"Replacement rule: Use META + APPROVED PROPOSAL to fill every placeholder. If information is missing, write 'TBD' (never guess).",

		"Status rule: Use meta.status_slug for {{STATUS_SLUG}} and meta.status_color for {{STATUS_COLOR}}.",
		"License rule: Use meta.license for {{LICENSE}}.",
		"Clone rule: Use meta.clone_command for {{CLONE_COMMAND}}.",
		"Demo rule: If no demo link or file is provided, set {{DEMO_EMBED}} to 'TBD'.",

		"Summary rule: 1-2 sentences that state the problem and the goal to solve it.",
		"Success criteria rule: Provide 3 concise, measurable bullets.",

		"Output ONLY the completed README in Markdown (no code fences, no commentary).",
	}, "\n")

	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	input := "TEMPLATE:\n" + string(tpl) + "\n\nMETA:\n" + string(metaJSON) + "\n\nAPPROVED PROPOSAL:\n" + proposal

	client := openai.NewClient(option.WithAPIKey(apiKey))
	resp, err := client.Responses.New(context.TODO(), responses.ResponseNewParams{
		Model:        model,
		Instructions: openai.String(instructions),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(input),
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai error: %w", err)
	}

	md := strings.TrimSpace(resp.OutputText())
	if md == "" {
		return "", fmt.Errorf("empty model output")
	}

	return md, nil
}

func randID(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func downloadID(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	return strings.TrimSuffix(name, ext)
}

func scheduleCleanup(id string) {
	cleanupMu.Lock()
	if t, ok := cleanupTimers[id]; ok {
		t.Stop()
	}

	var t *time.Timer
	t = time.AfterFunc(cleanupDelay, func() {
		cleanupMu.Lock()
		if cleanupTimers[id] != t {
			cleanupMu.Unlock()
			return
		}
		delete(cleanupTimers, id)
		cleanupMu.Unlock()
		removeGeneratedFiles(id)
	})
	cleanupTimers[id] = t
	cleanupMu.Unlock()
}

func removeGeneratedFiles(id string) {
	for _, ext := range []string{".md", ".pdf"} {
		path := filepath.Join(outDir, id+ext)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Printf("cleanup: remove %s failed: %v", path, err)
		}
	}
}
