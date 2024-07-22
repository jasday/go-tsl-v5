package client

import (
	"fmt"
	"net"

	"github.com/jasday/go-tsl-v5/pkg/server"
	"github.com/jasday/go-tsl-v5/pkg/tally"
)

type Option func(*Client) error

type Client struct {
	Protocol server.Protocol
}

func (c *Client) SendTally(t tally.Tally, conn net.Conn) {
	switch c.Protocol {
	case server.UDP:
		conn.Write(t.Bytes())
	}

	fmt.Println("Attempted to send tally with unknown protocol")
}
