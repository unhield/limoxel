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

// ActiveProcesses returns the currently tracked active process count.
func (r *ResourceLimiter) ActiveProcesses() uint32 {
	return atomic.LoadUint32(&r.activeProcs)
}

// CheckProcessSpawn verifies whether another child process may be spawned under current limits.
func (r *ResourceLimiter) CheckProcessSpawn() error {
	r.mu.RLock()
	max := r.limits.MaxProcesses
	r.mu.RUnlock()

	current := atomic.LoadUint32(&r.activeProcs)
	if max > 0 && current >= max {
		return fmt.Errorf("%w: active processes (%d) reached configured limit (%d)",
			ErrResourceExhausted, current, max)
	}

	return nil
}

// ReserveProcess atomically checks and reserves a child process slot.
func (r *ResourceLimiter) ReserveProcess() error {
	r.mu.RLock()
	max := r.limits.MaxProcesses
	r.mu.RUnlock()

	for {
		current := atomic.LoadUint32(&r.activeProcs)
		if max > 0 && current >= max {
			return fmt.Errorf("%w: active processes (%d) reached configured limit (%d)",
				ErrResourceExhausted, current, max)
		}
		if atomic.CompareAndSwapUint32(&r.activeProcs, current, current+1) {
			return nil
		}
	}
}

// IncrementProcesses increments the active process count.
func (r *ResourceLimiter) IncrementProcesses() {
	for {
		current := atomic.LoadUint32(&r.activeProcs)
		if atomic.CompareAndSwapUint32(&r.activeProcs, current, current+1) {
			return
		}
	}
}

// DecrementProcesses decrements the active process count, atomically preventing underflow.
func (r *ResourceLimiter) DecrementProcesses() {
	for {
		current := atomic.LoadUint32(&r.activeProcs)
		if current == 0 {
			return // Already zero; never underflow to uint32 max
		}
		if atomic.CompareAndSwapUint32(&r.activeProcs, current, current-1) {
			return
		}
	}
}

// ReleaseProcess releases a previously reserved process slot (alias for DecrementProcesses).
func (r *ResourceLimiter) ReleaseProcess() {
	r.DecrementProcesses()
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

// StartupTimeout returns the configured startup handshake timeout.
func (r *ResourceLimiter) StartupTimeout() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.limits.StartupTimeout <= 0 {
		return 5 * time.Second
	}
	return r.limits.StartupTimeout
}

// ShutdownTimeout returns the configured graceful shutdown timeout.
func (r *ResourceLimiter) ShutdownTimeout() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.limits.ShutdownTimeout <= 0 {
		return 3 * time.Second
	}
	return r.limits.ShutdownTimeout
}
