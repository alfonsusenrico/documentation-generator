package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var placeholderRe = regexp.MustCompile(`\{\{[A-Z0-9_]+\}\}`)

type Manager struct {
	Root string
}

func NewManager(root string) *Manager {
	return &Manager{Root: root}
}

func (m *Manager) Read(relPath string) (string, error) {
	path := filepath.Join(m.Root, relPath)
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", relPath, err)
	}
	return string(b), nil
}

func Apply(template string, values map[string]string) string {
	if len(values) == 0 {
		return template
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		pairs = append(pairs, "{{"+k+"}}", values[k])
	}
	replacer := strings.NewReplacer(pairs...)
	return replacer.Replace(template)
}

func FindPlaceholders(input string) []string {
	matches := placeholderRe.FindAllString(input, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	for _, m := range matches {
		seen[m] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for m := range seen {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

func EnsureNoPlaceholders(input string) error {
	missing := FindPlaceholders(input)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("unresolved placeholders: %s", strings.Join(missing, ", "))
}
