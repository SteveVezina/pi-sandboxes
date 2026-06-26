package policy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/policy"
)

func TestDefaultPolicy_Fields(t *testing.T) {
	p := policy.DefaultPolicy()

	if p.Filesystem.HostHomeMount {
		t.Error("Expected hostHomeMount to be false by default")
	}
	if p.Filesystem.Workspace != "read-write" {
		t.Errorf("Expected workspace 'read-write', got '%s'", p.Filesystem.Workspace)
	}
	if p.Process.MaxProcesses != 256 {
		t.Errorf("Expected maxProcesses 256, got %d", p.Process.MaxProcesses)
	}
	if p.Process.DefaultTimeout != 120 {
		t.Errorf("Expected defaultTimeout 120, got %d", p.Process.DefaultTimeout)
	}
	if p.Process.MaxOutput != 8*1024*1024 {
		t.Errorf("Expected maxOutput 8MiB, got %d", p.Process.MaxOutput)
	}
	if p.Network.Mode != "restricted" {
		t.Errorf("Expected network mode 'restricted', got '%s'", p.Network.Mode)
	}
	if p.Secrets.Env != "deny-by-default" {
		t.Errorf("Expected env 'deny-by-default', got '%s'", p.Secrets.Env)
	}
}

func TestDefaultPolicy_NetworkDeny(t *testing.T) {
	p := policy.DefaultPolicy()

	// Should deny cloud metadata
	found := false
	for _, ip := range p.Network.Deny {
		if ip == "169.254.169.254" {
			found = true
		}
	}
	if !found {
		t.Error("Expected 169.254.169.254 in deny list")
	}
}

func TestDefaultPolicy_NetworkAllow(t *testing.T) {
	p := policy.DefaultPolicy()

	// Should allow known registries
	expected := []string{"github.com", "pypi.org", "proxy.golang.org"}
	for _, host := range expected {
		found := false
		for _, h := range p.Network.Allow {
			if h == host {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected %s in allow list", host)
		}
	}
}

func TestDefaultPolicy_Secrets(t *testing.T) {
	p := policy.DefaultPolicy()

	if p.Secrets.SSHAgent != "opt-in" {
		t.Errorf("Expected sshAgent 'opt-in', got '%s'", p.Secrets.SSHAgent)
	}
	if p.Secrets.GitCredentials != "brokered" {
		t.Errorf("Expected gitCredentials 'brokered', got '%s'", p.Secrets.GitCredentials)
	}
}

func TestValidate(t *testing.T) {
	p := policy.DefaultPolicy()
	if err := p.Validate(); err != nil {
		t.Fatalf("Default policy should be valid: %v", err)
	}

	// Invalid: maxProcesses = 0
	p2 := *p
	p2.Process.MaxProcesses = 0
	if err := p2.Validate(); err == nil {
		t.Error("Expected error for maxProcesses=0")
	}

	// Invalid: defaultTimeout = 0
	p3 := *p
	p3.Process.DefaultTimeout = 0
	if err := p3.Validate(); err == nil {
		t.Error("Expected error for defaultTimeout=0")
	}

	// Invalid: maxOutput = 0
	p4 := *p
	p4.Process.MaxOutput = 0
	if err := p4.Validate(); err == nil {
		t.Error("Expected error for maxOutput=0")
	}
}

func TestIsNeverMounted(t *testing.T) {
	tests := []struct {
		path     string
		never    bool
	}{
		{"/var/run/docker.sock", true},
		{"/", true},
		{"/proc", true},
		{"/sys", true},
		{"/workspace", false},
		{"/artifacts", false},
		{"/home/user", false},
	}

	for _, tc := range tests {
		result := policy.IsNeverMounted(tc.path)
		if result != tc.never {
			t.Errorf("IsNeverMounted(%q) = %v, want %v", tc.path, result, tc.never)
		}
	}
}

func TestOverride_CannotRelaxMaxProcesses(t *testing.T) {
	base := policy.DefaultPolicy()
	override := &policy.Override{
		MaxProcesses: ptrInt(512), // Try to increase beyond default
	}

	result := override.ApplyMerge(base)
	if result.Process.MaxProcesses != 256 {
		t.Errorf("Expected maxProcesses to stay at 256, got %d", result.Process.MaxProcesses)
	}

	// Decrease should be allowed
	override2 := &policy.Override{
		MaxProcesses: ptrInt(128),
	}
	result2 := override2.ApplyMerge(base)
	if result2.Process.MaxProcesses != 128 {
		t.Errorf("Expected maxProcesses 128, got %d", result2.Process.MaxProcesses)
	}
}

func TestOverride_CannotRelaxMaxOutput(t *testing.T) {
	base := policy.DefaultPolicy()
	override := &policy.Override{
		MaxOutput: ptrInt64(16 * 1024 * 1024), // Try to increase beyond default
	}

	result := override.ApplyMerge(base)
	if result.Process.MaxOutput != 8*1024*1024 {
		t.Errorf("Expected maxOutput to stay at 8MiB, got %d", result.Process.MaxOutput)
	}
}

func TestOverride_CannotRelaxNetwork(t *testing.T) {
	base := policy.DefaultPolicy()
	override := &policy.Override{
		NetworkMode: ptrString("full"), // Try to relax from restricted
	}

	result := override.ApplyMerge(base)
	if result.Network.Mode != "restricted" {
		t.Errorf("Expected network mode to stay 'restricted', got '%s'", result.Network.Mode)
	}
}

func TestOverride_CannotMountHostHome(t *testing.T) {
	base := policy.DefaultPolicy()
	override := &policy.Override{
		HostHomeMount: ptrBool(true), // Try to enable host home mount
	}

	result := override.ApplyMerge(base)
	if result.Filesystem.HostHomeMount {
		t.Error("Expected hostHomeMount to stay false")
	}
}

func TestOverride_ValidDecrease(t *testing.T) {
	base := policy.DefaultPolicy()
	override := &policy.Override{
		MaxProcesses:   ptrInt(100),
		MaxOutput:      ptrInt64(4 * 1024 * 1024),
		DefaultTimeout: ptrInt64(60),
	}

	result := override.ApplyMerge(base)
	if result.Process.MaxProcesses != 100 {
		t.Errorf("Expected maxProcesses 100, got %d", result.Process.MaxProcesses)
	}
	if result.Process.MaxOutput != 4*1024*1024 {
		t.Errorf("Expected maxOutput 4MiB, got %d", result.Process.MaxOutput)
	}
	if result.Process.DefaultTimeout != 60 {
		t.Errorf("Expected defaultTimeout 60, got %d", result.Process.DefaultTimeout)
	}
}

func TestLoadPolicy_NoConfig(t *testing.T) {
	// Unset HOME so config file doesn't exist
	os.Unsetenv("HOME")
	p, err := policy.LoadPolicy()
	if err != nil {
		t.Fatalf("LoadPolicy should succeed with no config: %v", err)
	}
	if p.Process.MaxProcesses != 256 {
		t.Error("Expected default policy when no config exists")
	}
}

func TestLoadPolicy_InvalidYAML(t *testing.T) {
	piHome := filepath.Join(os.TempDir(), "pi-policy-test-"+randomID())
	os.MkdirAll(piHome, 0755)
	defer os.RemoveAll(piHome)

	os.WriteFile(filepath.Join(piHome, "config.yaml"), []byte("{invalid yaml!!!"), 0644)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", piHome)
	defer func() {
		os.Setenv("HOME", origHome)
	}()

	p, err := policy.LoadPolicy()
	if err != nil {
		t.Fatalf("LoadPolicy should fallback to defaults: %v", err)
	}
	if p.Process.MaxProcesses != 256 {
		t.Error("Expected default policy when config is invalid")
	}
}

func ptrInt(v int) *int       { return &v }
func ptrInt64(v int64) *int64 { return &v }
func ptrString(v string) *string { return &v }
func ptrBool(v bool) *bool   { return &v }

func randomID() string {
	b := []byte("abcdefghijklmnopqrstuvwxyz012345")
	n := len(b)
	result := make([]byte, 8)
	for i := range result {
		result[i] = b[i%n]
	}
	return string(result)
}
