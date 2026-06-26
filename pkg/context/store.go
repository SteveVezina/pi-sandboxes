// Package context provides CLI context management for local and remote daemons.
// Implements F22 per ADR-003.
package context

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Transport identifiers (ADR-003).
const (
	TransportUnix = "unix"
	TransportHTTP = "http"
	TransportSSH  = "ssh"
)

// Auth types (ADR-003).
const (
	AuthNone        = "none"
	AuthBearerToken = "bearer-token"
	AuthSSHAgent    = "ssh-agent"
)

// LocalContextName is the reserved name for the default local context.
const LocalContextName = "local"

// DefaultLocalTarget is the default daemon socket path.
const DefaultLocalTarget = "unix://~/.pi/sandboxd.sock"

// AuthConfig describes how to authenticate to a remote daemon.
// Per ADR-003, raw secrets are never written to disk; bearer tokens are
// referenced by environment variable name in TokenEnv.
type AuthConfig struct {
	Type string `yaml:"type"`
	// TokenEnv is the name of the env var holding the bearer token.
	// Empty for non-bearer auth types.
	TokenEnv string `yaml:"token_env,omitempty"`
	// SSHUser, SSHHost optional overrides for ssh-agent transport auth.
	SSHUser string `yaml:"ssh_user,omitempty"`
	SSHHost string `yaml:"ssh_host,omitempty"`
}

// Context describes one CLI daemon target.
type Context struct {
	Name      string     `yaml:"name"`
	Target    string     `yaml:"target"`
	Transport string     `yaml:"transport"`
	Auth      AuthConfig `yaml:"auth"`
}

// onDiskState is the YAML schema for ~/.pi/contexts.yaml.
type onDiskState struct {
	ActiveContext string    `yaml:"active_context"`
	Contexts      []Context `yaml:"contexts"`
}

// Store is the persistent context store.
type Store struct {
	path string
	mu   sync.Mutex
	st   onDiskState
}

// NewStore loads (or creates) a context store at the given path.
// If the file does not exist, the store is initialised with the default
// `local` context and an empty contexts list.
func NewStore(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("context store path is required")
	}
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.st = onDiskState{ActiveContext: LocalContextName}
			return nil
		}
		return fmt.Errorf("read context store: %w", err)
	}
	if len(data) == 0 {
		s.st = onDiskState{ActiveContext: LocalContextName}
		return nil
	}
	if err := yaml.Unmarshal(data, &s.st); err != nil {
		return fmt.Errorf("parse context store: %w", err)
	}
	if s.st.ActiveContext == "" {
		s.st.ActiveContext = LocalContextName
	}
	return nil
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create context store dir: %w", err)
	}
	data, err := yaml.Marshal(s.st)
	if err != nil {
		return fmt.Errorf("marshal context store: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write context store: %w", err)
	}
	return nil
}

// Validate checks a context against ADR-003 rules.
func Validate(ctx Context) error {
	if ctx.Name == "" {
		return fmt.Errorf("context name is required")
	}
	if ctx.Target == "" {
		return fmt.Errorf("context target is required")
	}
	switch ctx.Transport {
	case TransportUnix, TransportHTTP, TransportSSH:
	default:
		return fmt.Errorf("invalid transport %q: must be one of unix, http, ssh", ctx.Transport)
	}
	switch ctx.Auth.Type {
	case AuthNone, AuthBearerToken, AuthSSHAgent:
	default:
		return fmt.Errorf("invalid auth.type %q: must be one of none, bearer-token, ssh-agent", ctx.Auth.Type)
	}
	// ADR-003 transport/auth matrix:
	switch ctx.Transport {
	case TransportHTTP:
		if ctx.Auth.Type != AuthBearerToken {
			return fmt.Errorf("http transport requires auth.type=bearer-token (ADR-003)")
		}
		if ctx.Auth.TokenEnv == "" {
			return fmt.Errorf("bearer-token auth requires auth.token_env (no raw tokens on disk)")
		}
		if strings.ContainsAny(ctx.Auth.TokenEnv, " \t\n") {
			return fmt.Errorf("auth.token_env must be a single env var name")
		}
	case TransportSSH:
		if ctx.Auth.Type != AuthSSHAgent {
			return fmt.Errorf("ssh transport requires auth.type=ssh-agent (ADR-003)")
		}
	case TransportUnix:
		if ctx.Auth.Type != AuthNone {
			return fmt.Errorf("unix transport requires auth.type=none")
		}
	}
	return nil
}

// Create persists a new context.
func (s *Store) Create(ctx Context) error {
	if ctx.Name == LocalContextName {
		return fmt.Errorf("context name %q is reserved", LocalContextName)
	}
	if err := Validate(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.st.Contexts {
		if existing.Name == ctx.Name {
			return fmt.Errorf("context %q already exists", ctx.Name)
		}
	}
	s.st.Contexts = append(s.st.Contexts, ctx)
	return s.save()
}

// Get returns the named context, or the synthetic local context.
func (s *Store) Get(name string) (Context, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == LocalContextName {
		return localContext(), nil
	}
	for _, c := range s.st.Contexts {
		if c.Name == name {
			return c, nil
		}
	}
	return Context{}, fmt.Errorf("context %q not found", name)
}

// List returns all contexts including the synthetic local context.
func (s *Store) List() []Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Context{localContext()}
	out = append(out, s.st.Contexts...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Use switches the active context.
func (s *Store) Use(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == LocalContextName {
		s.st.ActiveContext = LocalContextName
		return s.save()
	}
	for _, c := range s.st.Contexts {
		if c.Name == name {
			s.st.ActiveContext = name
			return s.save()
		}
	}
	return fmt.Errorf("context %q not found", name)
}

// Active returns the currently-active context.
func (s *Store) Active() (Context, error) {
	s.mu.Lock()
	name := s.st.ActiveContext
	if name == "" {
		name = LocalContextName
	}
	s.mu.Unlock()
	return s.Get(name)
}

// ActiveName returns the name of the active context.
func (s *Store) ActiveName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.st.ActiveContext == "" {
		return LocalContextName
	}
	return s.st.ActiveContext
}

// Delete removes a context. Deleting the active context resets to local.
func (s *Store) Delete(name string) error {
	if name == LocalContextName {
		return fmt.Errorf("cannot delete reserved context %q", LocalContextName)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, c := range s.st.Contexts {
		if c.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("context %q not found", name)
	}
	s.st.Contexts = append(s.st.Contexts[:idx], s.st.Contexts[idx+1:]...)
	if s.st.ActiveContext == name {
		s.st.ActiveContext = LocalContextName
	}
	return s.save()
}

// Resolve returns the context to use for a command, honoring override if non-empty.
func (s *Store) Resolve(override string) (Context, error) {
	if override != "" {
		return s.Get(override)
	}
	return s.Active()
}

// localContext returns the synthetic always-present local context.
func localContext() Context {
	return Context{
		Name:      LocalContextName,
		Target:    DefaultLocalTarget,
		Transport: TransportUnix,
		Auth:      AuthConfig{Type: AuthNone},
	}
}

// DefaultPath returns the default context store path (~/.pi/contexts.yaml).
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}
	return filepath.Join(home, ".pi", "contexts.yaml")
}
