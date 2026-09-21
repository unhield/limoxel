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

// NewStorageLock creates a file lock for relPath rooted under baseDir.
// It enforces that relPath is a local relative path.
func NewStorageLock(baseDir, relPath string) (*ProcessFileLock, error) {
	if relPath == "" || relPath == "." || relPath == ".." {
		return nil, fmt.Errorf("%w: unsafe lock path: %s", ErrInvalidInput, relPath)
	}

	if strings.Contains(relPath, "..") {
		return nil, fmt.Errorf("%w: lock path cannot contain traversal sequences: %s", ErrInvalidInput, relPath)
	}

	if strings.HasPrefix(relPath, "/") || strings.HasPrefix(relPath, "\\") {
		return nil, fmt.Errorf("%w: lock path cannot be absolute or rooted: %s", ErrInvalidInput, relPath)
	}

	for i := 0; i < len(relPath); i++ {
		c := relPath[i]
		if c < 0x20 || c == 0x7f || c == ':' {
			return nil, fmt.Errorf("%w: lock path contains invalid character %q", ErrInvalidInput, c)
		}
	}

	if !filepath.IsLocal(relPath) {
		return nil, fmt.Errorf("%w: unsafe lock path: %s", ErrInvalidInput, relPath)
	}

	parts := strings.FieldsFunc(relPath, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: empty lock path", ErrInvalidInput)
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return nil, fmt.Errorf("%w: unsafe lock path component: %s", ErrInvalidInput, part)
		}
		if part[0] == '-' || part[0] == '.' || part[len(part)-1] == '.' {
			return nil, fmt.Errorf("%w: unsafe lock path component: %s", ErrInvalidInput, part)
		}
	}

	cleanBase := filepath.Clean(baseDir)
	if cleanBase == "" || cleanBase == "." {
		cleanBase = "marketplace_data"
	}

	targetPath := filepath.Join(cleanBase, relPath)
	return &ProcessFileLock{
		lockPath: targetPath + ".lock",
		timeout:  10 * time.Second,
	}, nil
}

// WithStorageLock executes fn while holding a cross-process lock on relPath under baseDir.
func WithStorageLock(baseDir, relPath string, fn func() error) error {
	lock, err := NewStorageLock(baseDir, relPath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(lock.lockPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create lock directory: %w", err)
	}

	if err := lock.Lock(); err != nil {
		return err
	}
	defer func() {
		_ = lock.Unlock()
	}()

	return fn()
}

// NewProcessFileLock creates a file lock for targetPath with a default 10s acquisition timeout.
func NewProcessFileLock(targetPath string) *ProcessFileLock {
	cleanPath := filepath.Clean(targetPath)
	return &ProcessFileLock{
		lockPath: cleanPath + ".lock",
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
			if _, writeErr := f.WriteString(payload); writeErr != nil {
				_ = f.Close()
				_ = os.Remove(l.lockPath)
				return fmt.Errorf("failed to write lock payload: %w", writeErr)
			}
			if closeErr := f.Close(); closeErr != nil {
				_ = os.Remove(l.lockPath)
				return fmt.Errorf("failed to close lock file: %w", closeErr)
			}
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
	err := os.Remove(l.lockPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// WithLock executes fn while holding the cross-process lock on lockPath.
func WithLock(lockPath string, fn func() error) error {
	cleanPath := filepath.Clean(lockPath)
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	lock := NewProcessFileLock(cleanPath)
	if err := lock.Lock(); err != nil {
		return err
	}
	defer func() {
		_ = lock.Unlock()
	}()

	return fn()
}
