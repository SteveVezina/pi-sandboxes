package session

import (
	"log"
	"os"
)

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
		meta, err := store.Get(id)
		if err != nil {
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
