package template

import (
	"crypto/sha256"
	"encoding/hex"

	"gopkg.in/yaml.v3"
)

// ContentDigest is a stable sha256 over the template's *definition* —
// what a sandbox built from it would contain. Provenance (source,
// lineage) and timestamps are excluded, so a fork or an imported copy of
// the same definition produces the same digest.
func (t *Template) ContentDigest() string {
	clone := *t
	clone.Source = nil
	clone.Lineage = nil
	clone.CreatedAt = ""
	clone.UpdatedAt = ""

	data, err := yaml.Marshal(&clone)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
