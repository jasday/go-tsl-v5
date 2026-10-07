package main

import (
	"context"
	"net"
	"testing"
	"time"

	tsl "github.com/jasday/go-tsl-v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func collect(n int) (Handler, <-chan []tsl.Packet) {
	ch := make(chan tsl.Packet, n)
	done := make(chan []tsl.Packet, 1)
	go func() {
		var got []tsl.Packet
		for p := range ch {
			got = append(got, p)
			if len(got) == n {
				done <- got
				return
			}
		}
	}()
	return func(p tsl.Packet, _ net.Addr) { ch <- p }, done
}

func waitFor[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
		panic("unreachable")
	}
}

func packets(n int) []tsl.Packet {
	ps := make([]tsl.Packet, n)
	for i := range ps {
		ps[i] = tsl.Packet{Displays: []tsl.Display{{Index: uint16(i), RightTally: tsl.LampRed, Text: "Cam"}}}
	}
	return ps
}

func TestServeUDP(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)

	want := packets(50)
	handle, got := collect(len(want))
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- ServeUDP(ctx, conn, handle) }()

	client, err := net.Dial("udp", conn.LocalAddr().String())
	require.NoError(t, err)
	defer client.Close()

	_, err = client.Write([]byte{0xFF, 0xFF, 0x00}) // malformed, skipped
	require.NoError(t, err)
	for i := range want {
		b, err := tsl.Marshal(&want[i])
		require.NoError(t, err)
		_, err = client.Write(b)
		require.NoError(t, err)
	}

	assert.Equal(t, want, waitFor(t, got), "packets are handled in order")
	cancel()
	assert.NoError(t, waitFor(t, served))
}

func TestServeTCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	want := packets(50)
	handle, got := collect(len(want))
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- ServeTCP(ctx, ln, handle) }()

	client, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	defer client.Close()

	_, err = client.Write([]byte{0xFE, 0x02, 0x08, 0x00, 0, 0, 0, 0, 1, 0, 0, 0}) // malformed, skipped
	require.NoError(t, err)
	enc := tsl.NewEncoder(client)
	for i := range want {
		require.NoError(t, enc.Encode(&want[i]))
	}

	assert.Equal(t, want, waitFor(t, got), "packets are handled in order")
	// Cancelling must also close the still-open client connection.
	cancel()
	assert.NoError(t, waitFor(t, served))
}

func TestRunUnknownProtocol(t *testing.T) {
	assert.Error(t, run(context.Background(), "sctp", ":0", nil))
}
