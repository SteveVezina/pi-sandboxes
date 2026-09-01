package template

import (
	"crypto/sha256"
	"encoding/hex"

	"gopkg.in/yaml.v3"
)

// ContentDigest is a stable sha256 over the template's definition,
// excluding volatile metadata (lineage and timestamps). Two templates
// with the same effective definition produce the same digest.
func (t *Template) ContentDigest() string {
	clone := *t
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
