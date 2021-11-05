package middleware

import (
	"fmt"
	"strings"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"golang.org/x/net/context"
)

var globalRegistry = NewRegistry()

// Registry is the interface for callers to get registered middleware.
type Registry interface {
	Register(name string, factoryMethod func(cfg *configv1.Middleware) (Middleware, error))
	Create(cfg *configv1.Middleware) (Middleware, error)
}

type middlewareRegistry struct {
	middleware map[string]func(cfg *configv1.Middleware) (Middleware, error)
}

// NewRegistry returns a new middleware registry.
func NewRegistry() Registry {
	return &middlewareRegistry{
		middleware: map[string]func(cfg *configv1.Middleware) (Middleware, error){},
	}
}

// Register registers one middleware.
func (p *middlewareRegistry) Register(name string, factoryMethod func(cfg *configv1.Middleware) (Middleware, error)) {
	p.middleware[createFullName(name)] = factoryMethod
}

// Create instantiates a middleware based on `cfg`.
func (p *middlewareRegistry) Create(cfg *configv1.Middleware) (Middleware, error) {
	if method, ok := p.getMiddleware(createFullName(cfg.Name)); ok {
		return method(cfg)
	}
	return nil, fmt.Errorf("Middleware %s has not been registered", cfg.Name)
}

func (p *middlewareRegistry) getMiddleware(name string) (func(*configv1.Middleware) (Middleware, error), bool) {
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
func Register(name string, factoryMethod func(cfg *configv1.Middleware) (Middleware, error)) {
	globalRegistry.Register(name, factoryMethod)
}

// Create instantiates a middleware based on `cfg`.
func Create(ctx context.Context, cfg *configv1.Middleware) (Middleware, error) {
	return globalRegistry.Create(cfg)
}
