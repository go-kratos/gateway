package manager

import (
	"context"

	pb "github.com/go-kratos/gateway/api/gateway/admin/v1"
	"google.golang.org/protobuf/proto"
)

// ConfigStore is a kv config store.
type ConfigStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Add(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
	Watch(ctx context.Context, prefix string, fn func(key string, value []byte)) error
}

// Manager is a proxy manager.
type Manager struct {
	pb.UnimplementedAdminServer

	store ConfigStore
}

// NewManager new a proxy manager.
func NewManager() *Manager {
	return &Manager{}
}

// AddService .
func (m *Manager) AddService(ctx context.Context, req *pb.AddServiceRequest) (*pb.AddServiceReply, error) {
	for _, s := range req.Services {
		v, err := proto.Marshal(s)
		if err != nil {
			return nil, err
		}
		if err := m.store.Add(ctx, s.Name, v); err != nil {
			return nil, err
		}
	}
	return &pb.AddServiceReply{}, nil
}

// DeleteService .
func (m *Manager) DeleteService(ctx context.Context, req *pb.DeleteServiceRequest) (*pb.DeleteServiceReply, error) {
	for _, name := range req.ServiceNames {
		if err := m.store.Delete(ctx, name); err != nil {
			return nil, err
		}
	}
	return &pb.DeleteServiceReply{}, nil
}

// ListService .
func (m *Manager) ListService(context.Context, *pb.DeleteServiceRequest) (*pb.ListServiceReply, error) {
	return nil, nil
}
