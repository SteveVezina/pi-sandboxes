package sandbox

import (
	"log"
	"time"
)

// TTLChecker runs a background goroutine that checks for expired sandboxes.
type TTLChecker struct {
	store    *Store
	interval time.Duration
	stopCh   chan struct{}
}

// NewTTLChecker creates a new TTL checker.
// interval is how often to check (default: 60s).
func NewTTLChecker(store *Store, interval time.Duration) *TTLChecker {
	if interval == 0 {
		interval = 60 * time.Second
	}
	return &TTLChecker{
		store:    store,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the background TTL checking goroutine.
func (c *TTLChecker) Start() {
	go c.run()
}

// Stop signals the background goroutine to exit.
func (c *TTLChecker) Stop() {
	close(c.stopCh)
}

func (c *TTLChecker) run() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.check()
		case <-c.stopCh:
			return
		}
	}
}

func (c *TTLChecker) check() {
	ids, err := c.store.List()
	if err != nil {
		log.Printf("sandbox: list sandboxes for TTL check: %v", err)
		return
	}

	for _, id := range ids {
		meta, err := c.store.Get(id)
		if err != nil {
			log.Printf("sandbox: get sandbox %s for TTL check: %v", id, err)
			continue
		}

		// TTL of 0 means infinite
		if meta.TTL == 0 {
			continue
		}

		expiry := meta.LastUsedAt.Add(time.Duration(meta.TTL) * time.Second)
		if time.Now().After(expiry) {
			log.Printf("sandbox: TTL expired for %s (last used: %s)", id, meta.LastUsedAt.Format(time.RFC3339))
			// Transition to destroying (idempotent from any state)
			_ = c.store.UpdateState(id, StateDestroying)
		}
	}
}
