package preference_service

import "sync"

// shortcut: in-process per-user lock (a single API instance); with several
// instances move the read-merge-write inside the repository's advisory-locked tx.
var userLocks sync.Map

func lockUser(userID string) func() {
	m, _ := userLocks.LoadOrStore(userID, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}
