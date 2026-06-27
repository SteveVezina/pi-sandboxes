package session

import (
	"time"

	"github.com/google/uuid"
)

// State represents the lifecycle state of a sandbox session.
type State string

const (
	StateCreating   State = "CREATING"
	StateWarm       State = "WARM"
	StateExecuting  State = "EXECUTING"
	StateDestroying State = "DESTROYING"
	StateDestroyed  State = "DESTROYED"
)

// Meta is the persistent metadata for a sandbox session.
type Meta struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Template      string    `json:"template"`
	Mode          string    `json:"mode"`
	State         State     `json:"state"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updatedAt"`
	TTL           int       `json:"ttl_seconds"`
	LastUsedAt    time.Time `json:"last_used_at"`
	Workspace     string    `json:"workspace"`
	WorkspaceMode string    `json:"workspace_mode"`
	Artifacts     string    `json:"artifacts"`
	Snapshots     []string  `json:"snapshots"`
}

// NewMeta creates a new session metadata with default values.
func NewMeta(name, template, mode string) *Meta {
	return &Meta{
		ID:            uuid.New().String(),
		Name:          name,
		Template:      template,
		Mode:          mode,
		State:         StateCreating,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		TTL:           7200, // default 2 hours
		WorkspaceMode: "copy",
		Snapshots:     []string{},
	}
}
