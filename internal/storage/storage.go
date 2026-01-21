package storage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"proposal/internal/util"
)

type Store struct {
	OutDir       string
	CleanupDelay time.Duration
	mu           sync.Mutex
	timers       map[string]*time.Timer
}

type ProposalSummary struct {
	ID          string    `json:"id"`
	Label       string    `json:"label"`
	ProjectName string    `json:"project_name"`
	ClientOwner string    `json:"client_or_owner"`
	Date        string    `json:"date"`
	Version     string    `json:"version"`
	PDFReady    bool      `json:"pdf_ready"`
	ModTime     time.Time `json:"-"`
}

type ProposalMeta struct {
	ProjectName string
	ClientOwner string
	PreparedBy  string
	Date        string
	Version     string
}

func NewStore(outDir string, cleanupDelay time.Duration) *Store {
	return &Store{
		OutDir:       outDir,
		CleanupDelay: cleanupDelay,
		timers:       map[string]*time.Timer{},
	}
}

func (s *Store) SaveProposal(md string) (string, error) {
	id := util.RandID(8)
	mdPath := filepath.Join(s.OutDir, id+".md")
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) UpdateProposal(id, md string) error {
	path := filepath.Join(s.OutDir, id+".md")
	return os.WriteFile(path, []byte(md), 0o644)
}

func (s *Store) ReadProposal(id string) (string, error) {
	path := filepath.Join(s.OutDir, id+".md")
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Store) ListProposals() ([]ProposalSummary, error) {
	entries, err := os.ReadDir(s.OutDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ProposalSummary{}, nil
		}
		return nil, err
	}
	var list []ProposalSummary
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		id := strings.TrimSuffix(name, ".md")
		if !util.IsHexID(id) {
			continue
		}
		path := filepath.Join(s.OutDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		meta := ParseProposalMeta(string(content))
		pdfReady := fileExists(filepath.Join(s.OutDir, id+".pdf"))
		label := BuildProposalLabel(meta, id)
		modTime := time.Time{}
		if info, err := entry.Info(); err == nil {
			modTime = info.ModTime()
		}

		list = append(list, ProposalSummary{
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
	return list, nil
}

func (s *Store) RenderPDF(mdPath, pdfPath string) (bool, error) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		return false, fmt.Errorf("pandoc not found in PATH")
	}
	out, err := exec.Command(
		"pandoc", mdPath, "-o", pdfPath,
		"--pdf-engine=xelatex",
		"-V", "papersize=a4",
		"-V", "geometry:margin=18mm",
		"-V", "fontsize=11pt",
		"-V", "linestretch=1.5",
		"-V", "mainfont=DejaVu Serif",
		"--metadata", "title=",
	).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("pandoc failed: %v; output=%s", err, strings.TrimSpace(string(out)))
	}
	return true, nil
}

func (s *Store) ScheduleCleanup(id string) {
	s.mu.Lock()
	if t, ok := s.timers[id]; ok {
		t.Stop()
	}
	var t *time.Timer
	t = time.AfterFunc(s.CleanupDelay, func() {
		s.mu.Lock()
		if s.timers[id] != t {
			s.mu.Unlock()
			return
		}
		delete(s.timers, id)
		s.mu.Unlock()
		s.removeGeneratedFiles(id)
	})
	s.timers[id] = t
	s.mu.Unlock()
}

func (s *Store) removeGeneratedFiles(id string) {
	for _, ext := range []string{".md", ".pdf"} {
		path := filepath.Join(s.OutDir, id+ext)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			// ignore
		}
	}
}

func ParseProposalMeta(md string) ProposalMeta {
	lines := strings.Split(md, "\n")
	meta := ProposalMeta{}
	if len(lines) > 0 {
		title := strings.TrimSpace(strings.TrimSuffix(lines[0], "\r"))
		if strings.HasPrefix(title, "# Project Proposal") {
			rest := strings.TrimSpace(strings.TrimPrefix(title, "# Project Proposal"))
			rest = strings.TrimLeftFunc(rest, func(r rune) bool {
				return unicode.IsSpace(r) || unicode.IsPunct(r)
			})
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

func BuildProposalLabel(meta ProposalMeta, fallback string) string {
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

func EnsureMetaString(meta map[string]interface{}, key, fallback string) string {
	if v, ok := meta[key].(string); ok {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			meta[key] = trimmed
			return trimmed
		}
	}
	meta[key] = fallback
	return fallback
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
