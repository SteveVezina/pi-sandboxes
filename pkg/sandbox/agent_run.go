package sandbox

import (
	"fmt"
	"sync"
	"time"
)

// RunState represents the lifecycle state of an agent run.
type RunState string

const (
	RunStatePending    RunState = "PENDING"
	RunStateStarting   RunState = "STARTING"
	RunStateRunning    RunState = "RUNNING"
	RunStateCompleted  RunState = "COMPLETED"
	RunStateFailed     RunState = "FAILED"
	RunStateCancelled  RunState = "CANCELLED"
)

// AgentRun represents an agent run inside a sandbox.
type AgentRun struct {
	RunID       string       `json:"run_id"`
	SandboxID   string       `json:"sandbox_id"`
	AgentName   string       `json:"agent_name"`
	RepoURL     string       `json:"repo_url,omitempty"`
	Prompt      string       `json:"prompt,omitempty"`
	State       RunState     `json:"state"`
	ExitCode    int          `json:"exit_code,omitempty"`
	StartedAt   time.Time    `json:"started_at"`
	CompletedAt time.Time    `json:"completed_at,omitempty"`
	Error       string       `json:"error,omitempty"`
}

// AgentRunStore manages agent run state.
type AgentRunStore struct {
	mu      sync.RWMutex
	runs    map[string]*AgentRun
	sandbox map[string]string // sandbox_id → run_id
}

// NewAgentRunStore creates a new agent run store.
func NewAgentRunStore() *AgentRunStore {
	return &AgentRunStore{
		runs:    make(map[string]*AgentRun),
		sandbox: make(map[string]string),
	}
}

// Create starts a new agent run.
func (s *AgentRunStore) Create(runID, sandboxID, agentName, repoURL, prompt string) (*AgentRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if sandbox already has a running agent
	if existing, ok := s.sandbox[sandboxID]; ok {
		run := s.runs[existing]
		if run != nil && (run.State == RunStateRunning || run.State == RunStateStarting) {
			return nil, fmt.Errorf("sandbox %s already has a running agent run (%s)", sandboxID, existing)
		}
	}

	run := &AgentRun{
		RunID:       runID,
		SandboxID:   sandboxID,
		AgentName:   agentName,
		RepoURL:     repoURL,
		Prompt:      prompt,
		State:       RunStatePending,
		StartedAt:   time.Now(),
	}

	s.runs[runID] = run
	s.sandbox[sandboxID] = runID
	return run, nil
}

// Get retrieves an agent run by ID.
func (s *AgentRunStore) Get(runID string) (*AgentRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	run, ok := s.runs[runID]
	if !ok {
		return nil, fmt.Errorf("agent run %s not found", runID)
	}
	return run, nil
}

// GetBySandbox retrieves the active agent run for a sandbox.
func (s *AgentRunStore) GetBySandbox(sandboxID string) (*AgentRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runID, ok := s.sandbox[sandboxID]
	if !ok {
		return nil, fmt.Errorf("no agent run for sandbox %s", sandboxID)
	}
	return s.runs[runID], nil
}

// UpdateState transitions the run to a new state.
func (s *AgentRunStore) UpdateState(runID string, state RunState, exitCode int, err string) (*AgentRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, ok := s.runs[runID]
	if !ok {
		return nil, fmt.Errorf("agent run %s not found", runID)
	}

	run.State = state
	if exitCode != 0 {
		run.ExitCode = exitCode
	}
	if err != "" {
		run.Error = err
	}
	if state == RunStateCompleted || state == RunStateFailed || state == RunStateCancelled {
		run.CompletedAt = time.Now()
	}

	return run, nil
}

// List returns all agent runs.
func (s *AgentRunStore) List() []*AgentRun {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var runs []*AgentRun
	for _, run := range s.runs {
		runs = append(runs, run)
	}
	return runs
}
