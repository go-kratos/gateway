package middleware

import (
	"errors"
	"strings"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
)

var globalRegistry = NewRegistry()

// ErrNotFound is middleware not found.
var ErrNotFound = errors.New("Middleware has not been registered")

// Registry is the interface for callers to get registered middleware.
type Registry interface {
	Register(name string, factory Factory)
	Create(cfg *configv1.Middleware) (Middleware, error)
}

type middlewareRegistry struct {
	middleware map[string]Factory
}

// NewRegistry returns a new middleware registry.
func NewRegistry() Registry {
	return &middlewareRegistry{
		middleware: map[string]Factory{},
	}
}

// Register registers one middleware.
func (p *middlewareRegistry) Register(name string, factory Factory) {
	p.middleware[createFullName(name)] = factory
}

// Create instantiates a middleware based on `cfg`.
func (p *middlewareRegistry) Create(cfg *configv1.Middleware) (Middleware, error) {
	if method, ok := p.getMiddleware(createFullName(cfg.Name)); ok {
		return method(cfg)
	}
	return nil, ErrNotFound
}

func (p *middlewareRegistry) getMiddleware(name string) (Factory, bool) {
	nameLower := strings.ToLower(name)
	middlewareFn, ok := p.middleware[nameLower]
	if ok {
		return middlewareFn, true
	}
	return nil, false
}

func createFullName(name string) string {
	return strings.ToLower("gateway.middleware." + name)
}

// Register registers one middleware.
func Register(name string, factory Factory) {
	globalRegistry.Register(name, factory)
}

// Create instantiates a middleware based on `cfg`.
func Create(cfg *configv1.Middleware) (Middleware, error) {
	return globalRegistry.Create(cfg)
}
