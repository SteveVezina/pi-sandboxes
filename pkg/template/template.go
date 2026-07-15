package template

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Template represents a sandbox template definition.
type Template struct {
	Name    string            `yaml:"name"`
	Runtime string            `yaml:"runtime"`
	Base    string            `yaml:"base"`
	Tools   []string          `yaml:"tools"`
	Mounts  map[string]string `yaml:"mounts"`
	Caches  map[string]string `yaml:"caches"`
	Network string            `yaml:"network"`
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
