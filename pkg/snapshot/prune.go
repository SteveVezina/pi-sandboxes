package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// PruneOrphans removes content-addressed snapshot blobs under
// <piHome>/snapshots/content-addressed-store that no per-sandbox snapshot
// pointer references. Returns the number of content dirs removed.
func PruneOrphans(piHome string) (int, error) {
	referenced := map[string]bool{}

	sandboxes := filepath.Join(piHome, "sandboxes")
	entries, _ := os.ReadDir(sandboxes)
	for _, sb := range entries {
		snaps := filepath.Join(sandboxes, sb.Name(), "snapshots")
		snapEntries, _ := os.ReadDir(snaps)
		for _, sn := range snapEntries {
			data, err := os.ReadFile(filepath.Join(snaps, sn.Name(), "meta.json"))
			if err != nil {
				continue
			}
			var meta Meta
			if json.Unmarshal(data, &meta) == nil && meta.Hash != "" {
				referenced[meta.Hash] = true
			}
		}
	}

	cas := filepath.Join(piHome, "snapshots", "content-addressed-store")
	prefixes, err := os.ReadDir(cas)
	if err != nil {
		return 0, nil // nothing to prune
	}

	removed := 0
	for _, pfx := range prefixes {
		rest, _ := os.ReadDir(filepath.Join(cas, pfx.Name()))
		for _, r := range rest {
			hash := pfx.Name() + r.Name()
			if referenced[hash] {
				continue
			}
			if os.RemoveAll(filepath.Join(cas, pfx.Name(), r.Name())) == nil {
				removed++
			}
		}
	}
	return removed, nil
}
