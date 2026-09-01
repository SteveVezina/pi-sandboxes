package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Template represents a sandbox template definition.
//
// The first block is the original minimal schema and stays required. The
// rest is the F28 metadata extension — every field is optional so the
// built-in templates remain valid unchanged.
type Template struct {
	Name    string            `yaml:"name"`
	Runtime string            `yaml:"runtime"`
	Base    string            `yaml:"base"`
	Image   string            `yaml:"image,omitempty"`
	Tools   []string          `yaml:"tools"`
	Mounts  map[string]string `yaml:"mounts"`
	Caches  map[string]string `yaml:"caches,omitempty"`
	Network string            `yaml:"network"`

	// --- F28 metadata (all optional) ---
	Version        string            `yaml:"version,omitempty"`
	Summary        string            `yaml:"summary,omitempty"`
	Description    string            `yaml:"description,omitempty"`
	Tags           []string          `yaml:"tags,omitempty"`
	Source         *Source           `yaml:"source,omitempty"`
	Compatibility  *Compatibility    `yaml:"compatibility,omitempty"`
	NetworkDomains []string          `yaml:"networkDomains,omitempty"`
	Resources      *ResourceDefaults `yaml:"resources,omitempty"`
	Lineage        *Lineage          `yaml:"lineage,omitempty"`
	CreatedAt      string            `yaml:"createdAt,omitempty"`
	UpdatedAt      string            `yaml:"updatedAt,omitempty"`
}

// SourceType classifies where a template came from.
type SourceType string

const (
	SourceBuiltin  SourceType = "builtin"
	SourceLocal    SourceType = "local"
	SourceSnapshot SourceType = "snapshot"
	SourceImported SourceType = "imported"
)

// Source records a template's provenance.
type Source struct {
	Type       SourceType `yaml:"type"`
	Parent     string     `yaml:"parent,omitempty"`
	ForkedFrom string     `yaml:"forkedFrom,omitempty"`
	SnapshotOf string     `yaml:"snapshotOf,omitempty"` // sandbox ID when Type == snapshot
}

// Compatibility is a declared hint. The authoritative capability comes
// from each runtime driver's Probe/CapabilityReport (ADR-005).
type Compatibility struct {
	PiBox    string            `yaml:"piBox,omitempty"`
	Runtimes map[string]string `yaml:"runtimes,omitempty"` // mode -> supported | planned | unsupported
}

// ResourceDefaults are requested defaults, still subject to daemon policy.
type ResourceDefaults struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
	Disk   string `yaml:"disk,omitempty"`
}

// Lineage tracks fork/snapshot relationships and content digests.
type Lineage struct {
	Generation    int    `yaml:"generation"`
	ParentDigest  string `yaml:"parentDigest,omitempty"`
	ContentDigest string `yaml:"contentDigest,omitempty"`
}

// ImageMappings maps template base shorthands to fully qualified OCI image names.
var ImageMappings = map[string]string{
	"debian-slim":         "docker.io/library/debian:bookworm-slim",
	"debian":              "docker.io/library/debian:bookworm",
	"node":                "docker.io/library/node:22-bookworm",
	"python":              "docker.io/library/python:3.13-bookworm",
	"go":                  "docker.io/library/golang:1.24-bookworm",
	"rust":                "docker.io/library/rust:1.80-bookworm",
	"ubuntu":              "docker.io/library/ubuntu:24.04",
	"alpine":              "docker.io/library/alpine:3.20",
	"fedora":              "docker.io/library/fedora:40",
	"archlinux":           "docker.io/library/archlinux:latest",
	"centos":              "docker.io/library/centos:9",
	"rockylinux":          "docker.io/library/rockylinux:9",
	"almalinux":           "docker.io/library/almalinux:9",
	"void":                "docker.io/library/void:latest",
	"nixos":               "docker.io/library/nixos:latest",
	"garuda":              "docker.io/library/garuda:latest",
	"gentoo":              "docker.io/library/gentoo:latest",
	"slackware":           "docker.io/library/slackware:latest",
	"guix":                "docker.io/library/guix:latest",
	"guix-system":         "docker.io/library/guix-system:latest",
	"guix-sd":             "docker.io/library/guix-sd:latest",
	"guix-debian":         "docker.io/library/guix-debian:latest",
	"guix-ubuntu":         "docker.io/library/guix-ubuntu:latest",
	"guix-fedora":         "docker.io/library/guix-fedora:latest",
	"guix-centos":         "docker.io/library/guix-centos:latest",
	"guix-rocky":          "docker.io/library/guix-rocky:latest",
	"guix-almalinux":      "docker.io/library/guix-almalinux:latest",
	"guix-void":           "docker.io/library/guix-void:latest",
	"guix-nixos":          "docker.io/library/guix-nixos:latest",
	"guix-garuda":         "docker.io/library/guix-garuda:latest",
	"guix-gentoo":         "docker.io/library/guix-gentoo:latest",
	"guix-slackware":      "docker.io/library/guix-slackware:latest",
	"guix-guix":           "docker.io/library/guix-guix:latest",
	"guix-guix-system":    "docker.io/library/guix-guix-system:latest",
	"guix-guix-sd":        "docker.io/library/guix-guix-sd:latest",
	"guix-guix-debian":    "docker.io/library/guix-guix-debian:latest",
	"guix-guix-ubuntu":    "docker.io/library/guix-guix-ubuntu:latest",
	"guix-guix-fedora":    "docker.io/library/guix-guix-fedora:latest",
	"guix-guix-centos":    "docker.io/library/guix-guix-centos:latest",
	"guix-guix-rocky":     "docker.io/library/guix-guix-rocky:latest",
	"guix-guix-almalinux": "docker.io/library/guix-guix-almalinux:latest",
	"guix-guix-void":      "docker.io/library/guix-guix-void:latest",
	"guix-guix-nixos":     "docker.io/library/guix-guix-nixos:latest",
	"guix-guix-garuda":    "docker.io/library/guix-guix-garuda:latest",
	"guix-guix-gentoo":    "docker.io/library/guix-guix-gentoo:latest",
	"guix-guix-slackware": "docker.io/library/guix-guix-slackware:latest",
}

// ResolveImage resolves a template base shorthand to a fully qualified OCI image name.
// If the base is already a full image reference (contains "/" or ":"), it is returned as-is.
// If the base is a known shorthand, it is mapped to the corresponding image.
// Otherwise, it defaults to "docker.io/library/{base}:latest".
func ResolveImage(base string) string {
	// If base is empty, return a sensible default
	if base == "" {
		return "docker.io/library/debian:bookworm-slim"
	}

	// If base already looks like a full image reference, return as-is
	if strings.Contains(base, "/") || strings.Contains(base, ":") {
		return base
	}

	// Look up the shorthand mapping
	if img, ok := ImageMappings[base]; ok {
		return img
	}

	// Default to docker.io/library/{base}:latest
	return "docker.io/library/" + base + ":latest"
}

// ResolveTemplateImage resolves the OCI image for a template.
// If the template has an explicit Image field, it is used.
// Otherwise, the Base field is resolved using ResolveImage().
func ResolveTemplateImage(t *Template) string {
	if t.Image != "" {
		return t.Image
	}
	return ResolveImage(t.Base)
}

// DefaultTemplates returns the built-in templates.
func DefaultTemplates() map[string]string {
	return map[string]string{
		"base":        baseTemplate,
		"node":        nodeTemplate,
		"python":      pythonTemplate,
		"go":          goTemplate,
		"rust":        rustTemplate,
		"node-python": nodePythonTemplate,
		"polyglot":    polyglotTemplate,
	}
}

// Store manages template files.
type Store struct {
	templatesDir string
}

// NewStore creates a new template store.
func NewStore(templatesDir string) *Store {
	if templatesDir == "" {
		templatesDir = filepath.Join(os.Getenv("HOME"), ".pi-box", "templates")
	}
	return &Store{templatesDir: templatesDir}
}

// Dir returns the templates directory path.
func (s *Store) Dir() string {
	return s.templatesDir
}

// List returns all available template names.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.templatesDir)
	if err != nil {
		return nil, fmt.Errorf("read templates dir: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// Get loads a template by name.
func (s *Store) Get(name string) (*Template, error) {
	yamlPath := filepath.Join(s.templatesDir, name, "template.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("read template file: %w", err)
	}

	var t Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse template YAML: %w", err)
	}
	if t.Name == "" {
		t.Name = name
	}
	return &t, nil
}

// Create writes a template definition.
func (s *Store) Create(name string, t *Template) error {
	dir := filepath.Join(s.templatesDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create template dir: %w", err)
	}

	data, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal template: %w", err)
	}

	yamlPath := filepath.Join(dir, "template.yaml")
	if err := os.WriteFile(yamlPath, data, 0644); err != nil {
		return fmt.Errorf("write template file: %w", err)
	}

	// Record a local revision (F28 T28.2). Best-effort: a revision-store
	// failure must not fail the write.
	if err := s.appendRevision(name, t, data); err != nil {
		return fmt.Errorf("record revision: %w", err)
	}
	return nil
}

// Delete removes a template.
func (s *Store) Delete(name string) error {
	dir := filepath.Join(s.templatesDir, name)
	return os.RemoveAll(dir)
}

// InstallDefaults installs all built-in templates.
func (s *Store) InstallDefaults() error {
	defaults := DefaultTemplates()
	for name, yamlData := range defaults {
		yamlPath := filepath.Join(s.templatesDir, name, "template.yaml")
		if _, err := os.Stat(yamlPath); err == nil {
			continue
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("stat template %s: %w", name, err)
		}

		var t Template
		if err := yaml.Unmarshal([]byte(yamlData), &t); err != nil {
			return fmt.Errorf("parse default template %s: %w", name, err)
		}
		if err := s.Create(name, &t); err != nil {
			return fmt.Errorf("install template %s: %w", name, err)
		}
	}
	return nil
}
