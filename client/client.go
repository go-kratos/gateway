package client

// Client is a gateway client.
type Client interface {
	Invoke() error
}
