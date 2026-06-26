package bench

// SupportedModes returns the runtime modes the benchmark suite can target.
// Includes microvm per F20 / AC-23.7.
func SupportedModes() []string {
	return []string{"fast", "compat", "secure", "microvm"}
}

// IsSupportedMode reports whether mode is a recognized benchmark mode.
func IsSupportedMode(mode string) bool {
	for _, m := range SupportedModes() {
		if m == mode {
			return true
		}
	}
	return false
}
