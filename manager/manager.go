package manager

import (
	"context"

	pb "github.com/go-kratos/gateway/api/gateway/admin/v1"
)

// Manager is a proxy manager.
type Manager struct {
	pb.UnimplementedAdminServer
}

// NewManager new a proxy manager.
func NewManager() *Manager {
	return &Manager{}
}

// AddGateway .
func (m *Manager) AddGateway(context.Context, *pb.AddGatewayRequest) (*pb.AddGatewayReply, error) {
	return nil, nil
}

// DeleteGateway .
func (m *Manager) DeleteGateway(context.Context, *pb.DeleteGatewayRequest) (*pb.DeleteGatewayReply, error) {
	return nil, nil
}

// ListGateway .
func (m *Manager) ListGateway(context.Context, *pb.ListGatewayRequest) (*pb.ListGatewayReply, error) {
	return nil, nil
}

// AddService .
func (m *Manager) AddService(context.Context, *pb.AddServiceRequest) (*pb.AddServiceReply, error) {
	return nil, nil
}

// DeleteService .
func (m *Manager) DeleteService(context.Context, *pb.DeleteServiceRequest) (*pb.DeleteServiceReply, error) {
	return nil, nil
}

// ListService .
func (m *Manager) ListService(context.Context, *pb.DeleteServiceRequest) (*pb.ListServiceReply, error) {
	return nil, nil
}
