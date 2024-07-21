package server

import (
	"fmt"
	"net"

	"github.com/jasday/go-tsl-v5/pkg/tally"
)

type Protocol string

const (
	dle               int = 0xFE
	stx               int = 0x02
	maximumPacketSize int = 2048
	packetControlData int = 6 // 6 bytes of control data
)

const (
	TCP Protocol = "tcp"
	UCP Protocol = "ucp"
)

type Option func(*Server) error

type Server struct {
	Address               string
	Port                  int
	Protocol              Protocol
	EnforcePacketLength   bool
	EnforcedVersionNumber int
}

func (s *Server) Listen(callback func(tally tally.Tally, remoteAddr string)) error {
	switch s.Protocol {
	case UCP:
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

	p := make([]byte, maximumPacketSize)
	for {
		_, remoteaddr, err := ser.ReadFromUDP(p)
		if err != nil {
			fmt.Printf("error reading UDP packet %v", err)
			continue
		}
		go callback(*tally.FromBuffer(p), remoteaddr.String())
	}
}
