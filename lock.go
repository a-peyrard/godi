package godi

import "sync"

type LockManager struct {
	mu    sync.Mutex
	locks map[Name]*sync.Mutex
}

func NewLockManager() *LockManager {
	return &LockManager{
		locks: make(map[Name]*sync.Mutex),
	}
}

func (lm *LockManager) GetLockFor(name Name) *sync.Mutex {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lock, exists := lm.locks[name]; exists {
		return lock
	}

	lock := &sync.Mutex{}
	lm.locks[name] = lock
	return lock
}

func (lm *LockManager) ReleaseLock(name Name) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	delete(lm.locks, name)
}
