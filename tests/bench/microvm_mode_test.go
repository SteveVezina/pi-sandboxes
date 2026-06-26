package bench_test

import (
	"testing"

	"github.com/pi-sandbox/pi/pkg/bench"
)

func TestSuite_MicroVMModeRecognized(t *testing.T) {
	suite := bench.NewSuite("microvm", "node-python")
	if suite.Mode != "microvm" {
		t.Fatalf("suite.Mode = %q, want microvm", suite.Mode)
	}
}

func TestSupportedModes_IncludesMicroVM(t *testing.T) {
	modes := bench.SupportedModes()
	found := false
	for _, m := range modes {
		if m == "microvm" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("SupportedModes() = %v, want to contain 'microvm'", modes)
	}
}
