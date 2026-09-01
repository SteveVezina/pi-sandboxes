package template

import (
	"fmt"
	"path"
	"strings"
)

var validNetworkModes = map[string]bool{"none": true, "restricted": true, "open": true, "": true}

var validRuntimeCompat = map[string]bool{
	"supported": true, "planned": true, "unsupported": true, "": true,
}

var knownRuntimeModes = map[string]bool{
	"fast": true, "compat": true, "secure": true, "microvm": true, "auto": true,
}

// Validate checks a template against the schema and policy-relevant rules.
// It returns a list of human-readable problems; an empty list means valid.
func (t *Template) Validate() []string {
	var problems []string

	if strings.TrimSpace(t.Name) == "" {
		problems = append(problems, "name is required")
	}
	if t.Base == "" && t.Image == "" {
		problems = append(problems, "one of base or image is required")
	}

	if !validNetworkModes[t.Network] {
		problems = append(problems, fmt.Sprintf("network %q is not one of none|restricted|open", t.Network))
	}

	for name, dst := range t.Caches {
		if !path.IsAbs(dst) {
			problems = append(problems, fmt.Sprintf("cache %q destination %q must be an absolute path", name, dst))
		}
		if !strings.HasPrefix(dst, "/cache/") {
			problems = append(problems, fmt.Sprintf("cache %q destination %q should live under /cache/", name, dst))
		}
	}
	for name, dst := range t.Mounts {
		if !path.IsAbs(dst) {
			problems = append(problems, fmt.Sprintf("mount %q destination %q must be an absolute path", name, dst))
		}
	}

	for _, tool := range t.Tools {
		if strings.Contains(tool, ":") {
			parts := strings.SplitN(tool, ":", 2)
			if parts[0] == "" || parts[1] == "" {
				problems = append(problems, fmt.Sprintf("tool %q has an empty name or version", tool))
			}
		}
	}

	if t.Compatibility != nil {
		for mode, state := range t.Compatibility.Runtimes {
			if !knownRuntimeModes[mode] {
				problems = append(problems, fmt.Sprintf("compatibility.runtimes has unknown mode %q", mode))
			}
			if !validRuntimeCompat[state] {
				problems = append(problems, fmt.Sprintf("compatibility.runtimes[%s] = %q is not supported|planned|unsupported", mode, state))
			}
		}
	}

	if t.Source != nil {
		switch t.Source.Type {
		case SourceBuiltin, SourceLocal, SourceSnapshot, SourceImported, "":
		default:
			problems = append(problems, fmt.Sprintf("source.type %q is not builtin|local|snapshot|imported", t.Source.Type))
		}
		if t.Source.Type == SourceSnapshot && t.Source.SnapshotOf == "" {
			problems = append(problems, "source.type is snapshot but source.snapshotOf (sandbox ID) is empty")
		}
	}

	return problems
}
