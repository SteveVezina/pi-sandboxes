package secrets

// SSHAgent manages SSH agent forwarding for Git operations.
type SSHAgent struct {
	Enabled  bool
	Forward  bool
	Scopes   []string // Which commands can use the agent
}

// CanForward checks if SSH agent forwarding is allowed for a command.
func (s *SSHAgent) CanForward(command string) bool {
	if !s.Forward {
		return false
	}
	// Only allow for Git commands
	if command == "git" || len(s.Scopes) == 0 {
		return true
	}
	for _, scope := range s.Scopes {
		if scope == command {
			return true
		}
	}
	return false
}

// DefaultSSHAgent returns the default SSH agent config.
func DefaultSSHAgent() *SSHAgent {
	return &SSHAgent{
		Enabled:  false,
		Forward:  false,
		Scopes:   []string{"git"},
	}
}
