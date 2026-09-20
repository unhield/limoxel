package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrLockTimeout indicates timeout acquiring cross-process lock.
	ErrLockTimeout = errors.New("marketplace storage: timeout acquiring file lock")
)

// ProcessFileLock provides cooperative cross-process mutual exclusion on a file path.
type ProcessFileLock struct {
	lockPath string
	timeout  time.Duration
}

// NewProcessFileLock creates a file lock for targetPath with a default 10s acquisition timeout.
func NewProcessFileLock(targetPath string) *ProcessFileLock {
	return &ProcessFileLock{
		lockPath: targetPath + ".lock",
		timeout:  10 * time.Second,
	}
}

// SetTimeout sets the maximum duration to wait when acquiring the file lock.
func (l *ProcessFileLock) SetTimeout(d time.Duration) {
	if d > 0 {
		l.timeout = d
	}
}

// Lock acquires the lock exclusively, waiting up to the configured timeout.
func (l *ProcessFileLock) Lock() error {
	deadline := time.Now().Add(l.timeout)
	sleepInterval := 10 * time.Millisecond

	for {
		f, err := os.OpenFile(l.lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if err == nil {
			// Lock acquired. Write PID and timestamp
			payload := fmt.Sprintf("%d\n%d", os.Getpid(), time.Now().Unix())
			_, _ = f.WriteString(payload)
			_ = f.Close()
			return nil
		}

		// If lock file exists, check for stale lock (> 30s)
		if os.IsExist(err) {
			if data, readErr := os.ReadFile(l.lockPath); readErr == nil {
				lines := strings.Split(strings.TrimSpace(string(data)), "\n")
				if len(lines) >= 2 {
					if ts, parseErr := strconv.ParseInt(lines[1], 10, 64); parseErr == nil {
						if time.Now().Unix()-ts > 30 {
							// Stale lock from dead process: safely break lock
							_ = os.Remove(l.lockPath)
							continue
						}
					}
				}
			}
		}

		if time.Now().After(deadline) {
			return ErrLockTimeout
		}

		time.Sleep(sleepInterval)
		if sleepInterval < 100*time.Millisecond {
			sleepInterval *= 2
		}
	}
}

// Unlock releases the lock file.
func (l *ProcessFileLock) Unlock() error {
	return os.Remove(l.lockPath)
}

// WithLock executes fn while holding the cross-process lock on lockPath.
func WithLock(lockPath string, fn func() error) error {
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	lock := NewProcessFileLock(lockPath)
	if err := lock.Lock(); err != nil {
		return err
	}
	defer func() {
		_ = lock.Unlock()
	}()

	return fn()
}
