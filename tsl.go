package tsl

import (
	"github.com/jasday/go-tsl-v5/pkg/client"
	"github.com/jasday/go-tsl-v5/pkg/server"
)

func NewServer(addr string, options ...server.Option) (*server.Server, error) {
	// Default values
	fb := &server.Server{
		Address:               addr,
		Port:                  5900,
		Protocol:              server.UCP,
		EnforcePacketLength:   false,
		EnforcedVersionNumber: 0,
	}

	// Apply options
	for _, op := range options {
		err := op(fb)
		if err != nil {
			return nil, err
		}
	}
	return fb, nil
}

func OptionUsePort(p int) server.Option {
	return func(s *server.Server) error { s.Port = p; return nil }
}

func OptionUseTCP() server.Option {
	return func(s *server.Server) error { s.Protocol = server.TCP; return nil }
}

func OptionEnforcePacketLengthCheck() server.Option {
	return func(s *server.Server) error { s.EnforcePacketLength = true; return nil }
}

func OptionEnforceTslVersion(version int) server.Option {
	return func(s *server.Server) error { s.EnforcedVersionNumber = version; return nil }
}

func NewClient(addr string) *client.Client {
	return nil
}
