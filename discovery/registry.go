package discovery

import (
	"fmt"
	"net/url"

	"github.com/go-kratos/kratos/v2/registry"
)

var globalRegistry = NewRegistry()

type Factory func(dsn *url.URL) (registry.Discovery, error)

// Registry is the interface for callers to get registered middleware.
type Registry interface {
	Register(name string, factory Factory)
	Create(discoveryDSN string) (registry.Discovery, error)
}

type discoveryRegistry struct {
	discovery map[string]Factory
}

// NewRegistry returns a new middleware registry.
func NewRegistry() Registry {
	return &discoveryRegistry{
		discovery: map[string]Factory{},
	}
}

func (d *discoveryRegistry) Register(name string, factory Factory) {
	d.discovery[name] = factory
}

func (d *discoveryRegistry) Create(discoveryDSN string) (registry.Discovery, error) {
	if discoveryDSN == "" {
		return nil, fmt.Errorf("discoveryDSN is empty")
	}

	dsn, err := url.Parse(discoveryDSN)
	if err != nil {
		return nil, fmt.Errorf("parse discoveryDSN error: %s", err)
	}

	factory, ok := d.discovery[dsn.Scheme]
	if !ok {
		return nil, fmt.Errorf("discovery %s has not been registered", dsn.Scheme)
	}

	impl, err := factory(dsn)
	if err != nil {
		return nil, fmt.Errorf("create discovery error: %s", err)
	}
	return impl, nil
}

// Register registers one discovery.
func Register(name string, factory Factory) {
	globalRegistry.Register(name, factory)
}

// Create instantiates a discovery based on `discoveryDSN`.
func Create(discoveryDSN string) (registry.Discovery, error) {
	return globalRegistry.Create(discoveryDSN)
}
