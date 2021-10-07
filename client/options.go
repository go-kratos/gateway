package client

// callInfo contains all related configuration and information about an RPC.
type callInfo struct {
	labels map[string]string
}

// csAttempt implements a single transport stream attempt within a
// clientStream.
type csAttempt struct {
}

// CallOption configures a Call before it starts or extracts information from
// a Call after it completes.
type CallOption interface {
	// before is called before the call is sent to any server.  If before
	// returns a non-nil error, the RPC fails with that error.
	before(*callInfo) error

	// after is called after the call has completed.  after cannot return an
	// error, so any failures should be reported via output parameters.
	after(*callInfo, *csAttempt)
}

// LabelOption .
type LabelOption struct {
	Labels map[string]string
}

func (o LabelOption) before(c *callInfo) error {
	o.Labels = c.labels
	return nil
}

func (o LabelOption) after(*callInfo, *csAttempt) {}

// WithLabels .
func WithLabels(labels map[string]string) CallOption {
	return LabelOption{Labels: labels}
}
