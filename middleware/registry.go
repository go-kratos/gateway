package middleware

import (
	"errors"
	"strings"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/prometheus/client_golang/prometheus"
)

var LOG = log.NewHelper(log.With(log.GetLogger(), "source", "middleware"))
var globalRegistry = NewRegistry()
var _failedMiddlewareCreate = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "go",
	Subsystem: "gateway",
	Name:      "failed_middleware_create",
	Help:      "The total number of failed middleware create",
}, []string{"name", "required"})

func init() {
	prometheus.MustRegister(_failedMiddlewareCreate)
}

// ErrNotFound is middleware not found.
var ErrNotFound = errors.New("Middleware has not been registered")

// Registry is the interface for callers to get registered middleware.
type Registry interface {
	Register(name string, factory Factory)
	RegisterV2(name string, factory FactoryV2)
	Create(cfg *configv1.Middleware) (MiddlewareV2, error)
}

type middlewareRegistry struct {
	middleware map[string]FactoryV2
}

// NewRegistry returns a new middleware registry.
func NewRegistry() Registry {
	return &middlewareRegistry{
		middleware: map[string]FactoryV2{},
	}
}

// Register registers one middleware.
func (p *middlewareRegistry) Register(name string, factory Factory) {
	p.middleware[createFullName(name)] = wrapFactory(factory)
}

func (p *middlewareRegistry) RegisterV2(name string, factory FactoryV2) {
	p.middleware[createFullName(name)] = factory
}

// Create instantiates a middleware based on `cfg`.
func (p *middlewareRegistry) Create(cfg *configv1.Middleware) (MiddlewareV2, error) {
	if method, ok := p.getMiddleware(createFullName(cfg.Name)); ok {
		if cfg.Required {
			// If the middleware is required, it must be created successfully.
			instance, err := method(cfg)
			if err != nil {
				_failedMiddlewareCreate.WithLabelValues(cfg.Name, "true").Inc()
				LOG.Errorw(log.DefaultMessageKey, "Failed to create required middleware", "reason", "create_required_middleware_failed", "name", cfg.Name, "error", err, "config", cfg)
				return nil, err
			}
			return instance, nil
		}
		instance, err := method(cfg)
		if err != nil {
			_failedMiddlewareCreate.WithLabelValues(cfg.Name, "false").Inc()
			LOG.Errorw(log.DefaultMessageKey, "Failed to create optional middleware", "reason", "create_optional_middleware_failed", "name", cfg.Name, "error", err, "config", cfg)
			return EmptyMiddleware, nil
		}
		return instance, nil
	}
	return nil, ErrNotFound
}

func (p *middlewareRegistry) getMiddleware(name string) (FactoryV2, bool) {
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

// RegisterV2 registers one v2 middleware.
func RegisterV2(name string, factory FactoryV2) {
	globalRegistry.RegisterV2(name, factory)
}

// Create instantiates a middleware based on `cfg`.
func Create(cfg *configv1.Middleware) (MiddlewareV2, error) {
	return globalRegistry.Create(cfg)
}
