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
	Conn     net.Conn
	buf      []byte
}

func (c *Client) SendTally(t tally.Tally) {
	c.buf = make([]byte, 2)
	switch c.Protocol {
	case server.UDP:
		c.Conn.Write(t.Bytes(c.buf))
		return
	}

	fmt.Println("Attempted to send tally with unknown protocol")
}
