package main

import (
	"net"
	"testing"

	tsl "github.com/jasday/go-tsl-v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLamp(t *testing.T) {
	for in, want := range map[string]tsl.Lamp{"off": tsl.LampOff, "RED": tsl.LampRed, "Green": tsl.LampGreen, "amber": tsl.LampAmber} {
		got, err := parseLamp(in)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	_, err := parseLamp("blue")
	assert.Error(t, err)
}

func TestBuildPacket(t *testing.T) {
	p, err := buildPacket(1, 2, "Cam", "red", "green", "amber", 2, true)
	require.NoError(t, err)
	assert.Equal(t, &tsl.Packet{Unicode: true, Screen: 1, Displays: []tsl.Display{
		{Index: 2, LeftTally: tsl.LampRed, TextTally: tsl.LampGreen, RightTally: tsl.LampAmber, Brightness: 2, Text: "Cam"},
	}}, p)

	_, err = buildPacket(0x10000, 0, "", "off", "off", "off", 3, false)
	require.Error(t, err)
	_, err = buildPacket(0, 0, "", "off", "off", "off", 4, false)
	require.Error(t, err)
	_, err = buildPacket(0, 0, "", "off", "pink", "off", 3, false)
	assert.Error(t, err)
}

var sent = &tsl.Packet{Displays: []tsl.Display{{Index: 1, RightTally: tsl.LampRed, Text: "Cam 1"}}}

func TestSendUDP(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, send("udp", conn.LocalAddr().String(), sent))

	buf := make([]byte, tsl.MaxUDPPacketSize)
	n, _, err := conn.ReadFrom(buf)
	require.NoError(t, err)
	var got tsl.Packet
	require.NoError(t, tsl.Unmarshal(buf[:n], &got))
	assert.Equal(t, *sent, got)
}

func TestSendTCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	errs := make(chan error, 1)
	go func() { errs <- send("tcp", ln.Addr().String(), sent) }()

	conn, err := ln.Accept()
	require.NoError(t, err)
	defer conn.Close()
	var got tsl.Packet
	require.NoError(t, tsl.NewDecoder(conn).Decode(&got))
	assert.Equal(t, *sent, got)
	require.NoError(t, <-errs)
}

func TestSendErrors(t *testing.T) {
	require.Error(t, send("udp", "127.0.0.1:1", &tsl.Packet{Displays: []tsl.Display{{Brightness: 9}}}))
	assert.Error(t, send("bogus", "127.0.0.1:1", sent))
}
