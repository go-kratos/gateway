package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	*http.Server

	log *log.Helper
}

func New(log *log.Helper, handler http.Handler, addr string, timeout time.Duration, idleTimeout time.Duration) *Server {
	httpSrv := &http.Server{
		Addr: addr,
		Handler: h2c.NewHandler(handler, &http2.Server{
			IdleTimeout: idleTimeout,
		}),
		ReadTimeout:       timeout,
		ReadHeaderTimeout: timeout,
		WriteTimeout:      timeout,
		IdleTimeout:       idleTimeout,
	}
	return &Server{Server: httpSrv, log: log}
}

func (s *Server) Start(ctx context.Context) error {
	s.BaseContext = func(net.Listener) context.Context {
		return ctx
	}
	s.log.Infof("gateway server listening on %s", s.Addr)
	return s.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("gateway server stopping")
	return s.Shutdown(ctx)
}

func (s *Server) Endpoint() (*url.URL, error) {
	hostport, err := extract(s.Addr)
	if err != nil {
		return nil, err
	}
	return &url.URL{Scheme: "http", Host: hostport}, nil
}

// Extract returns a private addr and port.
func extract(hostPort string) (string, error) {
	addr, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return "", err
	}

	if len(addr) > 0 && (addr != "0.0.0.0" && addr != "[::]" && addr != "::") {
		return net.JoinHostPort(addr, port), nil
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, rawAddr := range addrs {
			var ip net.IP
			switch addr := rawAddr.(type) {
			case *net.IPAddr:
				ip = addr.IP
			case *net.IPNet:
				ip = addr.IP
			default:
				continue
			}
			if isValidIP(ip.String()) {
				return net.JoinHostPort(ip.String(), port), nil
			}
		}
	}
	return "", fmt.Errorf("no valid ip found")
}

func isValidIP(addr string) bool {
	ip := net.ParseIP(addr)
	return ip.IsGlobalUnicast() && !ip.IsInterfaceLocalMulticast()
}
