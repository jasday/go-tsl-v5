package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/jasday/go-tsl-v5"
)

type Protocol string

const (
	TCP Protocol = "tcp"
	UDP Protocol = "udp"
)

type Option func(*Server) error

type Server struct {
	Ctx                   context.Context
	Address               string
	Port                  int
	Protocol              Protocol
	EnforcePacketLength   bool
	EnforcedVersionNumber int
}

func NewServer(addr string, options ...Option) (*Server, error) {
	// Default values
	if addr == "localhost" {
		addr = "127.0.0.1"
	}
	svr := &Server{
		Address:               addr,
		Port:                  5900,
		Protocol:              UDP,
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

func OptionUsePort(p int) Option {
	return func(s *Server) error { s.Port = p; return nil }
}

func OptionUseTCP() Option {
	return func(s *Server) error { s.Protocol = TCP; return nil }
}

func OptionWithContext(ctx context.Context) Option {
	return func(s *Server) error { s.Ctx = ctx; return nil }
}

func OptionEnforcePacketLengthCheck() Option {
	return func(s *Server) error { s.EnforcePacketLength = true; return nil }
}

func OptionEnforceTslVersion(version int) Option {
	return func(s *Server) error { s.EnforcedVersionNumber = version; return nil }
}

func (s *Server) Listen(callback func(p tsl.Packet, remoteAddr string)) error {
	switch s.Protocol {
	case UDP:
		return s.listenUDP(callback)
	}

	return fmt.Errorf("unknown protocol received")
}

func (s *Server) listenUDP(callback func(p tsl.Packet, remoteAddr string)) error {
	addr := net.UDPAddr{
		Port: s.Port,
		IP:   net.ParseIP(s.Address),
	}

	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return err
	}

	p := make([]byte, tsl.MaxUDPPacketSize)
	for {
		select {
		case <-s.Ctx.Done():
			return ser.Close()
		default:
			ser.SetReadDeadline(time.Now().Add(time.Second * 1))
			_, remoteaddr, err := ser.ReadFromUDP(p)
			if err != nil {
				if e, ok := err.(net.Error); ok && e.Timeout() {
					// if the error is a timeout, just start again
					// This is so we can watch for context cancellations
					continue
				}
				fmt.Printf("error reading UDP packet %v", err)
				continue
			}
			var tally *tsl.Packet
			err = tsl.Unmarshal(p, tally)
			if err != nil {
				fmt.Printf("error unmarshalling udp tally %v", err)
				continue
			}

			if tally == nil {
				continue
			}
			go callback(*tally, remoteaddr.String())
		}
	}
}
