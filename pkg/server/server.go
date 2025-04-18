package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/jasday/go-tsl-v5/pkg/tally"
)

type Protocol string

const (
	dle               int = 0xFE
	stx               int = 0x02
	packetControlData int = 6 // 6 bytes of control data
)

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

func (s *Server) Listen(callback func(tally tally.Tally, remoteAddr string)) error {
	switch s.Protocol {
	case UDP:
		return s.listenUDP(callback)
	}

	return fmt.Errorf("unknown protocol received")
}

func (s *Server) listenUDP(callback func(tally tally.Tally, remoteAddr string)) error {
	addr := net.UDPAddr{
		Port: s.Port,
		IP:   net.ParseIP(s.Address),
	}

	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return err
	}

	p := make([]byte, tally.MaximumPacketSize)
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
			go callback(*tally.FromBuffer(p), remoteaddr.String())
		}
	}
}
