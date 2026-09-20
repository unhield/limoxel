package sandbox

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ResourceLimiter tracks and enforces runtime resource quotas.
type ResourceLimiter struct {
	mu            sync.RWMutex
	limits        ResourceLimits
	activeProcs   uint32
	peakMemory    uint64
	currentMemory uint64
}

// NewResourceLimiter constructs an initialized resource limiter.
func NewResourceLimiter(limits ResourceLimits) *ResourceLimiter {
	return &ResourceLimiter{
		limits: limits,
	}
}

// Limits returns the active resource limits.
func (r *ResourceLimiter) Limits() ResourceLimits {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.limits
}

// CheckProcessSpawn verifies whether another child process may be spawned under current limits.
func (r *ResourceLimiter) CheckProcessSpawn() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	current := atomic.LoadUint32(&r.activeProcs)
	if r.limits.MaxProcesses > 0 && current >= r.limits.MaxProcesses {
		return fmt.Errorf("%w: active processes (%d) reached configured limit (%d)",
			ErrResourceExhausted, current, r.limits.MaxProcesses)
	}

	return nil
}

// IncrementProcesses increments the active process count.
func (r *ResourceLimiter) IncrementProcesses() {
	atomic.AddUint32(&r.activeProcs, 1)
}

// DecrementProcesses decrements the active process count.
func (r *ResourceLimiter) DecrementProcesses() {
	current := atomic.LoadUint32(&r.activeProcs)
	if current > 0 {
		atomic.AddUint32(&r.activeProcs, ^uint32(0))
	}
}

// CheckMessageSize verifies that an incoming or outgoing IPC payload does not exceed the allowed size.
func (r *ResourceLimiter) CheckMessageSize(sizeBytes int) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	maxBytes := r.limits.MaxMessageBytes
	if maxBytes <= 0 {
		maxBytes = 16 * 1024 * 1024
	}

	if sizeBytes > maxBytes {
		return fmt.Errorf("%w: payload size (%d bytes) exceeds maximum limit (%d bytes)",
			ErrResourceExhausted, sizeBytes, maxBytes)
	}

	return nil
}

// RecordMemoryUsage updates current and peak observed memory consumption.
func (r *ResourceLimiter) RecordMemoryUsage(bytes uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.currentMemory = bytes
	if bytes > r.peakMemory {
		r.peakMemory = bytes
	}

	if r.limits.MaxMemoryBytes > 0 && bytes > r.limits.MaxMemoryBytes {
		return fmt.Errorf("%w: memory usage (%d bytes) exceeded maximum ceiling (%d bytes)",
			ErrResourceExhausted, bytes, r.limits.MaxMemoryBytes)
	}

	return nil
}

// WithExecutionTimeout wraps a parent context with the configured execution timeout.
func (r *ResourceLimiter) WithExecutionTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	r.mu.RLock()
	timeout := r.limits.ExecutionTimeout
	r.mu.RUnlock()

	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return context.WithTimeout(parent, timeout)
}
