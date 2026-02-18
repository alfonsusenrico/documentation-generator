package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"proposal/internal/ai"
	"proposal/internal/config"
	"proposal/internal/progress"
	"proposal/internal/project"
	"proposal/internal/scaffold"
	"proposal/internal/storage"
	"proposal/internal/templates"
	"proposal/internal/util"
)

type Server struct {
	cfg       config.Config
	templates *templates.Manager
	ai        *ai.Client
	store     *storage.Store
	scaffold  *scaffold.Generator
}

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
	ProposalID      string                 `json:"proposal_id"`
	Visibility      string                 `json:"visibility"`
	Stack           string                 `json:"stack"`
	Tier            string                 `json:"tier"`
	AutomationLevel string                 `json:"automation_level"`
	DeployMode      string                 `json:"deploy_mode"`
	Meta            map[string]interface{} `json:"meta"`
}

type initResp struct {
	ProjectPath  string   `json:"project_path"`
	RepoURL      string   `json:"repo_url"`
	RepoName     string   `json:"repo_name"`
	NextCommands []string `json:"next_commands"`
}

type updateProposalReq struct {
	Markdown string `json:"markdown"`
}

func NewServer(cfg config.Config, templates *templates.Manager, aiClient *ai.Client, store *storage.Store, scaffoldGen *scaffold.Generator) *Server {
	return &Server{
		cfg:       cfg,
		templates: templates,
		ai:        aiClient,
		store:     store,
		scaffold:  scaffoldGen,
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/proposal", s.handleProposal)
	mux.HandleFunc("/api/proposal/stream", s.handleProposalStream)
	mux.HandleFunc("/api/init", s.handleInit)
	mux.HandleFunc("/api/init/stream", s.handleInitStream)
	mux.HandleFunc("/api/proposals", s.handleProposals)
	mux.HandleFunc("/api/proposals/", s.handleProposalDetail)
	mux.HandleFunc("/api/agent/proposal", s.handleAgentProposal)
	mux.HandleFunc("/api/agent/init", s.handleAgentInit)
	mux.HandleFunc("/download/", s.handleDownload)
}

func (s *Server) handleProposal(w http.ResponseWriter, r *http.Request) {
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
	resp, err := s.generateProposal(req, nil)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			writeJSONError(w, apiErr.Status, apiErr.Message)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "proposal failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleProposalStream(w http.ResponseWriter, r *http.Request) {
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

	stream, err := newStreamWriter(w)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	totalSteps := 3
	if s.cfg.EnablePDF {
		totalSteps = 4
	}
	tracker := newProgressTracker(totalSteps, func(message, detail string, percent int) {
		_ = stream.Send(streamEvent{
			Type:    "progress",
			Message: message,
			Detail:  detail,
			Percent: percent,
		})
	})
	reporter := progress.ReporterFunc(func(ev progress.Event) {
		tracker.Step(ev.Message, ev.Detail)
	})

	resp, err := s.generateProposal(req, reporter)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			_ = stream.Send(streamEvent{Type: "error", Error: apiErr.Message})
			return
		}
		_ = stream.Send(streamEvent{Type: "error", Error: err.Error()})
		return
	}

	_ = stream.Send(streamEvent{Type: "result", Data: resp})
}

func (s *Server) generateProposal(req proposalReq, reporter progress.Reporter) (proposalResp, error) {
	if req.Meta == nil {
		req.Meta = map[string]interface{}{}
	}

	reportProgress(reporter, "Loading template", "Preparing proposal template.")
	tpl, err := s.templates.Read(filepath.Join("proposal", "PROPOSAL_TEMPLATE.md"))
	if err != nil {
		return proposalResp{}, apiError{Status: http.StatusInternalServerError, Message: "missing proposal template"}
	}

	// Defaults
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
	if s.cfg.PreparedBy != "" {
		if v, ok := req.Meta["your_name"].(string); !ok || strings.TrimSpace(v) == "" {
			req.Meta["your_name"] = s.cfg.PreparedBy
		}
	}

	reportProgress(reporter, "Generating proposal", "Summarizing the project plan.")
	md, err := s.ai.GenerateProposal(tpl, req.Meta, req.Plan, s.cfg.EnablePDF)
	if err != nil {
		return proposalResp{}, apiError{Status: http.StatusBadGateway, Message: err.Error()}
	}

	reportProgress(reporter, "Saving proposal", "Writing Markdown output.")
	id, err := s.store.SaveProposal(md)
	if err != nil {
		return proposalResp{}, apiError{Status: http.StatusInternalServerError, Message: "write md failed"}
	}

	projectName, _ := req.Meta["project_name"].(string)
	clientOwner, _ := req.Meta["client_or_owner"].(string)
	version, _ := req.Meta["version"].(string)

	baseName := "Proposal_" + util.SanitizeName(projectName) + "-" + util.SanitizeName(clientOwner) + "_" + util.SanitizeName(version)
	mdDownloadName := baseName + ".md"
	pdfDownloadName := baseName + ".pdf"

	pdfReady := false
	if s.cfg.EnablePDF {
		reportProgress(reporter, "Rendering PDF", "Building the PDF preview.")
		mdPath := filepath.Join(s.cfg.OutDir, id+".md")
		pdfPath := filepath.Join(s.cfg.OutDir, id+".pdf")
		if ok, err := s.store.RenderPDF(mdPath, pdfPath); err != nil {
			if strings.Contains(err.Error(), "pandoc not found") {
				return proposalResp{}, apiError{Status: http.StatusInternalServerError, Message: "ENABLE_PDF=1 but pandoc not found in PATH"}
			}
			log.Printf("pandoc: %v", err)
		} else {
			pdfReady = ok
		}
	}

	mdURL := "/download/" + id + ".md" + "?name=" + url.QueryEscape(mdDownloadName)
	pdfURL := "/download/" + id + ".pdf" + "?name=" + url.QueryEscape(pdfDownloadName)
	if s.cfg.PublicBaseURL != "" {
		mdURL = s.cfg.PublicBaseURL + mdURL
		pdfURL = s.cfg.PublicBaseURL + pdfURL
	}

	var out proposalResp
	out.ID = id
	out.Markdown = md
	out.PDFReady = pdfReady
	out.Downloads.MD = mdURL
	out.Downloads.PDF = pdfURL

	return out, nil
}

func (s *Server) handleProposals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	list, err := s.store.ListProposals()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "read out directory failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string][]storage.ProposalSummary{"items": list})
}

func (s *Server) handleProposalDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/proposals/")
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "..") || !util.IsHexID(id) {
		writeJSONError(w, http.StatusBadRequest, "invalid proposal id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleProposalDetailGet(w, r, id)
	case http.MethodPut:
		s.handleProposalUpdate(w, r, id)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleProposalDetailGet(w http.ResponseWriter, r *http.Request, id string) {
	content, err := s.store.ReadProposal(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "proposal not found")
		return
	}

	meta := storage.ParseProposalMeta(content)
	pdfReady := fileExists(filepath.Join(s.cfg.OutDir, id+".pdf"))

	baseName := "Proposal_" + util.SanitizeName(meta.ProjectName) + "-" + util.SanitizeName(meta.ClientOwner) + "_" + util.SanitizeName(meta.Version)
	mdName := baseName + ".md"
	pdfName := baseName + ".pdf"

	mdURL := "/download/" + id + ".md" + "?name=" + url.QueryEscape(mdName)
	pdfURL := "/download/" + id + ".pdf" + "?name=" + url.QueryEscape(pdfName)
	if s.cfg.PublicBaseURL != "" {
		mdURL = s.cfg.PublicBaseURL + mdURL
		pdfURL = s.cfg.PublicBaseURL + pdfURL
	}

	var out proposalDetailResp
	out.ID = id
	out.Markdown = strings.TrimSpace(content)
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

func (s *Server) handleProposalUpdate(w http.ResponseWriter, r *http.Request, id string) {
	var req updateProposalReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	md := strings.TrimSpace(req.Markdown)
	if md == "" {
		writeJSONError(w, http.StatusBadRequest, "missing markdown")
		return
	}

	if err := s.store.UpdateProposal(id, md); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "write md failed")
		return
	}

	pdfReady := false
	if s.cfg.EnablePDF {
		mdPath := filepath.Join(s.cfg.OutDir, id+".md")
		pdfPath := filepath.Join(s.cfg.OutDir, id+".pdf")
		if ok, err := s.store.RenderPDF(mdPath, pdfPath); err != nil {
			if strings.Contains(err.Error(), "pandoc not found") {
				writeJSONError(w, http.StatusInternalServerError, "ENABLE_PDF=1 but pandoc not found in PATH")
				return
			}
			log.Printf("pandoc: %v", err)
		} else {
			pdfReady = ok
		}
	}

	meta := storage.ParseProposalMeta(md)
	baseName := "Proposal_" + util.SanitizeName(meta.ProjectName) + "-" + util.SanitizeName(meta.ClientOwner) + "_" + util.SanitizeName(meta.Version)
	mdName := baseName + ".md"
	pdfName := baseName + ".pdf"

	mdURL := "/download/" + id + ".md" + "?name=" + url.QueryEscape(mdName)
	pdfURL := "/download/" + id + ".pdf" + "?name=" + url.QueryEscape(pdfName)
	if s.cfg.PublicBaseURL != "" {
		mdURL = s.cfg.PublicBaseURL + mdURL
		pdfURL = s.cfg.PublicBaseURL + pdfURL
	}

	resp := proposalResp{
		ID:       id,
		Markdown: md,
		PDFReady: pdfReady,
	}
	resp.Downloads.MD = mdURL
	resp.Downloads.PDF = pdfURL

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.cfg.InitToken != "" && r.Header.Get("X-Init-Token") != s.cfg.InitToken {
		writeJSONError(w, http.StatusUnauthorized, "invalid init token")
		return
	}

	var req initReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.normalizeInitRequest(&req); err != nil {
		if apiErr, ok := asAPIError(err); ok {
			writeJSONError(w, apiErr.Status, apiErr.Message)
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid init request")
		return
	}

	resp, err := s.initProject(req, nil)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			writeJSONError(w, apiErr.Status, apiErr.Message)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "init failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleInitStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.cfg.InitToken != "" && r.Header.Get("X-Init-Token") != s.cfg.InitToken {
		writeJSONError(w, http.StatusUnauthorized, "invalid init token")
		return
	}

	var req initReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.normalizeInitRequest(&req); err != nil {
		if apiErr, ok := asAPIError(err); ok {
			writeJSONError(w, apiErr.Status, apiErr.Message)
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid init request")
		return
	}

	stream, err := newStreamWriter(w)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	totalSteps := 2 + scaffold.DocFileCount() + 1 + 3
	tracker := newProgressTracker(totalSteps, func(message, detail string, percent int) {
		_ = stream.Send(streamEvent{
			Type:    "progress",
			Message: message,
			Detail:  detail,
			Percent: percent,
		})
	})
	reporter := progress.ReporterFunc(func(ev progress.Event) {
		tracker.Step(ev.Message, ev.Detail)
	})

	resp, err := s.initProject(req, reporter)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			_ = stream.Send(streamEvent{Type: "error", Error: apiErr.Message})
			return
		}
		_ = stream.Send(streamEvent{Type: "error", Error: err.Error()})
		return
	}

	_ = stream.Send(streamEvent{Type: "result", Data: resp})
}

func (s *Server) handleAgentProposal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.authorizeAgentRequest(w, r) {
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
	resp, err := s.generateProposal(req, nil)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			writeJSONError(w, apiErr.Status, apiErr.Message)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "proposal failed")
		return
	}

	log.Printf("agent_proposal_ok id=%s", resp.ID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleAgentInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.authorizeAgentRequest(w, r) {
		return
	}

	var req initReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.normalizeInitRequest(&req); err != nil {
		if apiErr, ok := asAPIError(err); ok {
			writeJSONError(w, apiErr.Status, apiErr.Message)
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid init request")
		return
	}

	resp, err := s.initProject(req, nil)
	if err != nil {
		if apiErr, ok := asAPIError(err); ok {
			writeJSONError(w, apiErr.Status, apiErr.Message)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "init failed")
		return
	}

	log.Printf("agent_init_ok repo=%s path=%s", resp.RepoName, resp.ProjectPath)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) authorizeAgentRequest(w http.ResponseWriter, r *http.Request) bool {
	if s.cfg.AgentToken == "" {
		writeJSONError(w, http.StatusForbidden, "agent api disabled: AGENT_TOKEN not set")
		return false
	}
	if r.Header.Get("X-Agent-Token") != s.cfg.AgentToken {
		writeJSONError(w, http.StatusUnauthorized, "invalid agent token")
		return false
	}
	return true
}

func (s *Server) normalizeInitRequest(req *initReq) error {
	req.ProposalID = strings.TrimSpace(req.ProposalID)
	if req.ProposalID == "" {
		return apiError{Status: http.StatusBadRequest, Message: "missing proposal_id"}
	}
	if !util.IsHexID(req.ProposalID) {
		return apiError{Status: http.StatusBadRequest, Message: "invalid proposal_id"}
	}
	req.Stack = strings.ToLower(strings.TrimSpace(req.Stack))
	if req.Stack == "" {
		return apiError{Status: http.StatusBadRequest, Message: "missing stack"}
	}
	req.Tier = strings.TrimSpace(req.Tier)
	if req.Tier == "" {
		return apiError{Status: http.StatusBadRequest, Message: "missing tier"}
	}

	req.Visibility = strings.ToLower(strings.TrimSpace(req.Visibility))
	if req.Visibility != "public" && req.Visibility != "private" {
		return apiError{Status: http.StatusBadRequest, Message: "visibility must be public or private"}
	}

	req.AutomationLevel = strings.ToLower(strings.TrimSpace(req.AutomationLevel))
	if req.AutomationLevel == "" {
		req.AutomationLevel = "repo_ci"
	}
	if req.AutomationLevel != "repo_only" && req.AutomationLevel != "repo_ci" && req.AutomationLevel != "repo_ci_cd" {
		return apiError{Status: http.StatusBadRequest, Message: "automation_level must be repo_only, repo_ci, or repo_ci_cd"}
	}

	req.DeployMode = strings.ToLower(strings.TrimSpace(req.DeployMode))
	if req.DeployMode == "" {
		req.DeployMode = "none"
	}
	if req.DeployMode != "none" && req.DeployMode != "ssh_compose" {
		return apiError{Status: http.StatusBadRequest, Message: "deploy_mode must be none or ssh_compose"}
	}
	if req.AutomationLevel != "repo_ci_cd" && req.DeployMode != "none" {
		return apiError{Status: http.StatusBadRequest, Message: "deploy_mode requires automation_level=repo_ci_cd"}
	}

	if req.Meta == nil {
		req.Meta = map[string]interface{}{}
	}

	return nil
}

func (s *Server) initProject(req initReq, reporter progress.Reporter) (initResp, error) {
	reportProgress(reporter, "Loading proposal", "Preparing inputs and templates.")
	proposalMD, err := s.store.ReadProposal(req.ProposalID)
	if err != nil {
		return initResp{}, apiError{Status: http.StatusNotFound, Message: "proposal not found"}
	}
	proposalMeta := storage.ParseProposalMeta(proposalMD)
	fillMetaIfEmpty(req.Meta, "project_name", proposalMeta.ProjectName)
	fillMetaIfEmpty(req.Meta, "client_or_owner", proposalMeta.ClientOwner)
	fillMetaIfEmpty(req.Meta, "date", proposalMeta.Date)
	fillMetaIfEmpty(req.Meta, "version", proposalMeta.Version)

	clientName := storage.EnsureMetaString(req.Meta, "client_or_owner", "Project Client")
	projectName := storage.EnsureMetaString(req.Meta, "project_name", "Project")
	clientSlug := util.Slugify(clientName, "client")
	projectSlug := util.Slugify(projectName, "project")
	repoName := clientSlug + "-" + projectSlug
	owner := defaultOwner(s.cfg.GithubOwner)

	reportProgress(reporter, "Validating environment", "Checking tools, repo, and target path.")
	if _, err := project.Preflight(project.PreflightRequest{
		Owner:        owner,
		RepoName:     repoName,
		Visibility:   req.Visibility,
		ClientSlug:   clientSlug,
		ProjectSlug:  projectSlug,
		GitUserName:  s.cfg.GitUserName,
		GitUserEmail: s.cfg.GitUserEmail,
	}); err != nil {
		return initResp{}, apiError{Status: http.StatusBadRequest, Message: err.Error()}
	}

	result, err := s.scaffold.Generate(scaffold.Request{
		Proposal:        proposalMD,
		Meta:            req.Meta,
		Stack:           req.Stack,
		Tier:            req.Tier,
		Visibility:      req.Visibility,
		AutomationLevel: req.AutomationLevel,
		DeployMode:      req.DeployMode,
		GithubOwner:     owner,
		PreparedBy:      s.cfg.PreparedBy,
		Reporter:        reporter,
	})
	if err != nil {
		return initResp{}, apiError{Status: http.StatusBadRequest, Message: err.Error()}
	}

	files := make([]project.File, 0, len(result.Files))
	for _, f := range result.Files {
		files = append(files, project.File{Path: f.Path, Content: f.Content})
	}

	initResult, err := project.Init(project.InitRequest{
		Owner:        defaultOwner(s.cfg.GithubOwner),
		RepoName:     result.RepoName,
		Visibility:   req.Visibility,
		ClientSlug:   result.ClientSlug,
		ProjectSlug:  result.ProjectSlug,
		GitUserName:  s.cfg.GitUserName,
		GitUserEmail: s.cfg.GitUserEmail,
		Files:        files,
		Reporter:     reporter,
	})
	if err != nil {
		return initResp{}, apiError{Status: http.StatusBadGateway, Message: "init failed: " + err.Error()}
	}

	outResp := initResp{
		ProjectPath: initResult.ProjectPath,
		RepoURL:     initResult.RepoURL,
		RepoName:    initResult.RepoName,
		NextCommands: []string{
			"git clone " + initResult.RepoURL,
			"cd " + initResult.RepoName,
		},
	}

	return outResp, nil
}

func reportProgress(reporter progress.Reporter, message, detail string) {
	if reporter == nil {
		return
	}
	reporter.Report(progress.Event{Message: message, Detail: detail})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/download/")
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\\`) {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	path := filepath.Join(s.cfg.OutDir, name)
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

	if id := util.DownloadID(name); id != "" {
		s.store.ScheduleCleanup(id)
	}
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func defaultOwner(owner string) string {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return "alfonsusenrico"
	}
	return owner
}

func fillMetaIfEmpty(meta map[string]interface{}, key, value string) {
	if value == "" {
		return
	}
	if v, ok := meta[key].(string); ok && strings.TrimSpace(v) != "" {
		return
	}
	meta[key] = value
}

type apiError struct {
	Status  int
	Message string
}

func (e apiError) Error() string {
	return e.Message
}

func asAPIError(err error) (apiError, bool) {
	var apiErr apiError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return apiError{}, false
}
