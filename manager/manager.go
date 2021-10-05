package manager

// Manager is a proxy manager.
type Manager interface {
	// gateway
	AddGateway() error
	DelGateway() error
	ListGateway() error
	// service
	AddService() error
	DelService() error
	ListService() error
}
