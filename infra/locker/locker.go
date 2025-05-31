// locker/locker.go
package locker

import "sync"

type Locker struct {
	mu           sync.Mutex
	inProcessMap map[int64]bool
}

func New() *Locker {
	return &Locker{
		inProcessMap: make(map[int64]bool),
	}
}

// MarkAsProcessing adds a log ID to the in-memory map.
func (l *Locker) MarkAsProcessing(logID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.inProcessMap[logID] = true
}

// IsProcessing checks if a log ID is already being processed.
func (l *Locker) IsProcessing(logID int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.inProcessMap[logID]
}

func (l *Locker) Unlock(logID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.inProcessMap, logID)
}
