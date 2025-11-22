package client

import (
	"errors"
	"net"

	"github.com/jasday/go-tsl-v5"
	"github.com/jasday/go-tsl-v5/example/server"
)

type Option func(*Client) error

type Client struct {
	Protocol server.Protocol
	Conn     net.Conn
	buf      []byte
}

func NewClient(addr string, conn net.Conn, options ...Option) (*Client, error) {
	client := &Client{
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

func (c *Client) SendTally(t *tsl.Tally) error {
	c.buf = make([]byte, 2)
	switch c.Protocol {
	case server.UDP:
		buf, err := tsl.Marshal(t)
		if err != nil {
			return err
		}
		c.Conn.Write(buf)
	default:
		return errors.New("attempted to send tally with unsupported protocol")
	}

	return nil
}
