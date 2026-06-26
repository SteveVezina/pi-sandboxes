package secrets

// Broker manages secret exposure for sandbox sessions.
type Broker struct {
	secrets map[string]*Secret
}

// Secret represents a single secret.
type Secret struct {
	Name     string   `json:"name"`
	Value    string   `json:"-"`
	ExposeTo []string `json:"exposeTo"` // Which processes can access this secret
	Source   string   `json:"source"`   // "env", "ssh-agent", "token-helper"
}

// NewBroker creates a new secret broker.
func NewBroker() *Broker {
	return &Broker{
		secrets: make(map[string]*Secret),
	}
}

// Add registers a secret.
func (b *Broker) Add(name, value string, exposeTo []string, source string) {
	b.secrets[name] = &Secret{
		Name:     name,
		Value:    value,
		ExposeTo: exposeTo,
		Source:   source,
	}
}

// Get returns a secret by name.
func (b *Broker) Get(name string) (*Secret, bool) {
	s, ok := b.secrets[name]
	return s, ok
}

// List returns all secret names.
func (b *Broker) List() []string {
	var names []string
	for name := range b.secrets {
		names = append(names, name)
	}
	return names
}

// Remove deletes a secret.
func (b *Broker) Remove(name string) {
	delete(b.secrets, name)
}

// Has checks if a secret exists.
func (b *Broker) Has(name string) bool {
	_, ok := b.secrets[name]
	return ok
}
