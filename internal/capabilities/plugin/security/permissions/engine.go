package permissions

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrPermissionDenied indicates that the evaluated action was not authorized.
	ErrPermissionDenied = errors.New("security: permission denied")
)

// ViolationHandler receives notification of an unauthorized access attempt.
type ViolationHandler func(req PermissionRequest, res EvaluationResult)

// Engine evaluates permission requests against registered plugin policies using fail-closed semantics.
type Engine struct {
	mu        sync.RWMutex
	policies  map[string]*PluginPolicy
	handlers  []ViolationHandler
	handlerMu sync.RWMutex
}

// NewEngine constructs an initialized, thread-safe permission engine.
func NewEngine() *Engine {
	return &Engine{
		policies: make(map[string]*PluginPolicy),
	}
}

// SetPolicy assigns or updates the security policy for a specific plugin.
func (e *Engine) SetPolicy(policy *PluginPolicy) {
	if policy == nil || policy.PluginID == "" {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policies[policy.PluginID] = policy
}

// RemovePolicy removes the policy for a plugin upon uninstallation or deactivation.
func (e *Engine) RemovePolicy(pluginID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.policies, pluginID)
}

// GetPolicy retrieves the current policy for a plugin.
func (e *Engine) GetPolicy(pluginID string) (*PluginPolicy, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	p, ok := e.policies[pluginID]
	return p, ok
}

// OnViolation registers an audit observer for permission violations.
func (e *Engine) OnViolation(h ViolationHandler) {
	if h == nil {
		return
	}
	e.handlerMu.Lock()
	defer e.handlerMu.Unlock()
	e.handlers = append(e.handlers, h)
}

// Evaluate checks whether a request is authorized, returning an EvaluationResult.
func (e *Engine) Evaluate(ctx context.Context, req PermissionRequest) EvaluationResult {
	if ctx != nil && ctx.Err() != nil {
		return EvaluationResult{
			Request:  req,
			Decision: DecisionDenied,
			Reason:   fmt.Sprintf("context canceled: %v", ctx.Err()),
		}
	}

	if req.PluginID == "" {
		res := EvaluationResult{
			Request:  req,
			Decision: DecisionDenied,
			Reason:   "missing plugin identity in permission request",
		}
		e.notifyViolation(req, res)
		return res
	}

	e.mu.RLock()
	policy, exists := e.policies[req.PluginID]
	e.mu.RUnlock()

	if !exists {
		res := EvaluationResult{
			Request:  req,
			Decision: DecisionDenied,
			Reason:   fmt.Sprintf("no security policy configured for plugin %s (fail-closed)", req.PluginID),
		}
		e.notifyViolation(req, res)
		return res
	}

	// Match against granted permissions
	for _, grant := range policy.Grants {
		if grant.Matches(req.Action, req.Resource) {
			return EvaluationResult{
				Request:   req,
				Decision:  DecisionGranted,
				Reason:    fmt.Sprintf("authorized by grant %s", grant.Action),
				MatchedOn: grant.Action,
			}
		}
	}

	// Fail closed if no grant matched
	res := EvaluationResult{
		Request:  req,
		Decision: DecisionDenied,
		Reason:   fmt.Sprintf("action '%s' on resource '%s' is not granted by plugin policy", req.Action, req.Resource),
	}
	e.notifyViolation(req, res)
	return res
}

// Check evaluates the request and returns an error if permission is denied.
func (e *Engine) Check(ctx context.Context, req PermissionRequest) error {
	res := e.Evaluate(ctx, req)
	if !res.Allowed() {
		return fmt.Errorf("%w: %s", ErrPermissionDenied, res.Reason)
	}
	return nil
}

func (e *Engine) notifyViolation(req PermissionRequest, res EvaluationResult) {
	e.handlerMu.RLock()
	handlers := make([]ViolationHandler, len(e.handlers))
	copy(handlers, e.handlers)
	e.handlerMu.RUnlock()

	for _, h := range handlers {
		h(req, res)
	}
}
