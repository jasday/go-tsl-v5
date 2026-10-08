package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"

	tsl "github.com/jasday/go-tsl-v5"
)

// Handler is called for each packet received. Packets from one sender are
// handled in order; with TCP, separate connections are handled concurrently.
type Handler func(p tsl.Packet, from net.Addr)

// ServeUDP reads one packet per datagram from conn until ctx is cancelled.
// Malformed packets are logged and skipped. conn is closed on return.
func ServeUDP(ctx context.Context, conn net.PacketConn, handle Handler) error {
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	// Read up to the largest possible datagram so oversized packets are still parsed.
	buf := make([]byte, 65535)
	for {
		n, from, err := conn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		var p tsl.Packet
		if err := tsl.Unmarshal(buf[:n], &p); err != nil {
			log.Printf("%v: %v", from, err)
			continue
		}
		handle(p, from)
	}
}

// ServeTCP accepts connections on ln and reads framed packets from each
// until ctx is cancelled. ln and all connections are closed on return.
func ServeTCP(ctx context.Context, ln net.Listener, handle Handler) error {
	defer ln.Close()
	stop := context.AfterFunc(ctx, func() { _ = ln.Close() })
	defer stop()

	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			serveConn(ctx, conn, handle)
		}()
	}
}

func serveConn(ctx context.Context, conn net.Conn, handle Handler) {
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	dec := tsl.NewDecoder(conn)
	for {
		var p tsl.Packet
		err := dec.Decode(&p)
		switch {
		case err == nil:
			handle(p, conn.RemoteAddr())
		case errors.Is(err, tsl.ErrMalformed), errors.Is(err, tsl.ErrShortPacket):
			log.Printf("%v: %v", conn.RemoteAddr(), err)
		default:
			if !errors.Is(err, io.EOF) && ctx.Err() == nil {
				log.Printf("%v: %v", conn.RemoteAddr(), err)
			}
			return
		}
	}
}
