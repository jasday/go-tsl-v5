package tsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testPacket = Packet{
	Displays: []Display{{Brightness: 3, Text: "Test"}},
}

var testPacketBytes = []byte{0xe, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xc0, 0x0, 0x4, 0x0, 0x54, 0x65, 0x73, 0x74}

func TestMarshal(t *testing.T) {
	got, err := Marshal(&testPacket)
	require.NoError(t, err)
	assert.Equal(t, testPacketBytes, got)
}

func TestUnmarshal(t *testing.T) {
	var got Packet
	require.NoError(t, Unmarshal(testPacketBytes, &got))
	assert.Equal(t, testPacket, got)
}
