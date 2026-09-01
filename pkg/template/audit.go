package template

import "log/slog"

// audit records a mutating template-library operation (AC-35.11). It never
// carries credential material — templates have no secret fields, and
// bundles are content-addressed.
func audit(op, name, detail string) {
	slog.Info("template op", "op", op, "template", name, "detail", detail)
}
