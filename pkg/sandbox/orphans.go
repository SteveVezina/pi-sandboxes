package sandbox

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
)

// sessionIDRegex matches UUID-style session IDs.
var sessionIDRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// OrphanCleanup scans for sessions in non-terminal states with no
// corresponding backend process and marks them as DESTROYED.
func OrphanCleanup(store *Store, baseDir string) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("session: orphan cleanup: read base dir: %v", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		id := entry.Name()

		// Skip non-session directories (e.g., var, temp, etc.)
		if !sessionIDRegex.MatchString(id) {
			continue
		}

		meta, err := store.Get(id)
		if err != nil {
			// Check if it's a "not found" error (missing meta.json)
			if isNotFound(err) {
				// Skip missing metadata
				continue
			}
			log.Printf("session: orphan cleanup: get session %s: %v", id, err)
			continue
		}

		// Only clean up non-terminal states
		if meta.State != StateCreating && meta.State != StateExecuting {
			continue
		}

		// Mark as destroyed (workspace/artifacts preserved)
		log.Printf("session: orphan cleanup: marking %s (%s) as DESTROYED", id, meta.Name)
		if err := store.UpdateState(id, StateDestroyed); err != nil {
			log.Printf("session: orphan cleanup: update state for %s: %v", id, err)
		}
	}
}

// isNotFound checks if an error is a "file not found" error.
func isNotFound(err error) bool {
	return os.IsNotExist(err) || filepath.Base(err.Error()) == "no such file or directory"
}
