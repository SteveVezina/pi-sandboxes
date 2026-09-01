package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Revision is one recorded state of a local template.
type Revision struct {
	N      int    `json:"n"`
	Time   string `json:"time"`
	Digest string `json:"digest"`
}

func (s *Store) revisionsDir(name string) string {
	return filepath.Join(s.templatesDir, name, "revisions")
}

// appendRevision snapshots the just-written template YAML as revision N+1.
func (s *Store) appendRevision(name string, t *Template, yamlData []byte) error {
	dir := s.revisionsDir(name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	history, _ := s.History(name)
	n := 1
	if len(history) > 0 {
		n = history[0].N + 1
	}

	if err := os.WriteFile(filepath.Join(dir, strconv.Itoa(n)+".yaml"), yamlData, 0o644); err != nil {
		return err
	}

	rev := Revision{N: n, Time: time.Now().UTC().Format(time.RFC3339), Digest: t.ContentDigest()}
	history = append([]Revision{rev}, history...)
	idx, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "index.json"), idx, 0o644)
}

// History returns a template's local revisions, newest first.
func (s *Store) History(name string) ([]Revision, error) {
	data, err := os.ReadFile(filepath.Join(s.revisionsDir(name), "index.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return []Revision{}, nil
		}
		return nil, err
	}
	var revs []Revision
	if err := json.Unmarshal(data, &revs); err != nil {
		return nil, err
	}
	sort.Slice(revs, func(i, j int) bool { return revs[i].N > revs[j].N })
	return revs, nil
}

// GetRevision loads a specific revision of a template.
func (s *Store) GetRevision(name string, n int) (*Template, error) {
	data, err := os.ReadFile(filepath.Join(s.revisionsDir(name), strconv.Itoa(n)+".yaml"))
	if err != nil {
		return nil, fmt.Errorf("revision %d of %q not found", n, name)
	}
	var t Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse revision: %w", err)
	}
	return &t, nil
}

// Rollback restores a template to a prior revision. The rollback is itself
// recorded as a new revision.
func (s *Store) Rollback(name string, n int) (*Template, error) {
	prior, err := s.GetRevision(name, n)
	if err != nil {
		return nil, err
	}
	cur, err := s.Get(name)
	if err != nil {
		return nil, err
	}

	restored := *prior
	now := time.Now().UTC().Format(time.RFC3339)
	restored.UpdatedAt = now
	gen := 0
	if cur.Lineage != nil {
		gen = cur.Lineage.Generation
	}
	restored.Lineage = &Lineage{
		Generation:   gen + 1,
		ParentDigest: cur.ContentDigest(),
	}
	restored.Lineage.ContentDigest = restored.ContentDigest()

	if err := s.Create(name, &restored); err != nil {
		return nil, err
	}
	return &restored, nil
}

// ResolveRef loads a template by "name" or "name@N" (a revision).
func (s *Store) ResolveRef(ref string) (*Template, error) {
	if name, rev, ok := strings.Cut(ref, "@"); ok {
		n, err := strconv.Atoi(rev)
		if err != nil {
			return nil, fmt.Errorf("bad revision in %q", ref)
		}
		return s.GetRevision(name, n)
	}
	return s.Get(ref)
}

// Diff renders a line-oriented diff of two templates' canonical YAML.
func Diff(left, right *Template) string {
	l, _ := yaml.Marshal(left)
	r, _ := yaml.Marshal(right)
	return lineDiff(string(l), string(r))
}

// lineDiff is a minimal line-by-line diff — enough to see what changed
// between two templates without a diff-library dependency.
func lineDiff(a, b string) string {
	al := strings.Split(strings.TrimRight(a, "\n"), "\n")
	bl := strings.Split(strings.TrimRight(b, "\n"), "\n")
	inB := map[string]bool{}
	for _, line := range bl {
		inB[line] = true
	}
	inA := map[string]bool{}
	for _, line := range al {
		inA[line] = true
	}

	var out strings.Builder
	for _, line := range al {
		if !inB[line] {
			out.WriteString("- " + line + "\n")
		}
	}
	for _, line := range bl {
		if !inA[line] {
			out.WriteString("+ " + line + "\n")
		}
	}
	if out.Len() == 0 {
		return "(no differences)\n"
	}
	return out.String()
}
