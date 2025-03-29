package tsl

import (
	"context"
	"net"

	"github.com/jasday/go-tsl-v5/pkg/client"
	"github.com/jasday/go-tsl-v5/pkg/server"
)

func NewServer(addr string, options ...server.Option) (*server.Server, error) {
	// Default values
	if addr == "localhost" {
		addr = "127.0.0.1"
	}
	svr := &server.Server{
		Address:               addr,
		Port:                  5900,
		Protocol:              server.UDP,
		EnforcePacketLength:   false,
		EnforcedVersionNumber: 0,
	}

	// Apply options
	for _, op := range options {
		err := op(svr)
		if err != nil {
			return nil, err
		}
	}
	return svr, nil
}

func OptionUsePort(p int) server.Option {
	return func(s *server.Server) error { s.Port = p; return nil }
}

func OptionUseTCP() server.Option {
	return func(s *server.Server) error { s.Protocol = server.TCP; return nil }
}

func OptionWithContext(ctx context.Context) server.Option {
	return func(s *server.Server) error { s.Ctx = ctx; return nil }
}

func OptionEnforcePacketLengthCheck() server.Option {
	return func(s *server.Server) error { s.EnforcePacketLength = true; return nil }
}

func OptionEnforceTslVersion(version int) server.Option {
	return func(s *server.Server) error { s.EnforcedVersionNumber = version; return nil }
}

func NewClient(addr string, conn net.Conn, options ...client.Option) (*client.Client, error) {
	client := &client.Client{
		Protocol: server.UDP,
		Conn:     conn,
	}

	// Apply options
	for _, op := range options {
		err := op(client)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}
