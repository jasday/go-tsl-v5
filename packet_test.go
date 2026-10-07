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

func TestUnmarshalDecodesFields(t *testing.T) {
	in := []byte{
		0x14, 0x00, // PBC 20
		0x00,       // VER
		0x00,       // FLAGS
		0x34, 0x12, // SCREEN 0x1234
		// DMSG 1: index 0xFFFF, RH red, text green, LH amber, brightness 2, text "AB"
		0xFF, 0xFF, 0b10_11_10_01, 0x00, 0x02, 0x00, 'A', 'B',
		// DMSG 2: index 5, control data set
		0x05, 0x00, 0x00, 0x80,
		// DMSG 3: index 6, empty text
		0x06, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	in[0] = byte(len(in) - 2)

	var got Packet
	require.NoError(t, Unmarshal(in, &got))
	assert.Equal(t, Packet{
		Screen: 0x1234,
		Displays: []Display{
			{Index: BroadcastIndex, RightTally: LampRed, TextTally: LampGreen, LeftTally: LampAmber, Brightness: 2, Text: "AB"},
			{Index: 5, ControlData: true},
			{Index: 6},
		},
	}, got)
}

func TestUnmarshalFlags(t *testing.T) {
	tests := []struct {
		flags         byte
		unicode       bool
		screenControl bool
	}{
		{0x00, false, false},
		{0x01, true, false},
		{0x02, false, true},
		{0x03, true, true},
		{0xFC, false, false}, // reserved bits ignored
	}
	for _, tt := range tests {
		var got Packet
		require.NoError(t, Unmarshal([]byte{4, 0, 0, tt.flags, 0, 0}, &got))
		assert.Equal(t, tt.unicode, got.Unicode, "flags %#x", tt.flags)
		assert.Equal(t, tt.screenControl, got.ScreenControl, "flags %#x", tt.flags)
	}
}

func TestUnmarshalHeaderOnly(t *testing.T) {
	var got Packet
	require.NoError(t, Unmarshal([]byte{4, 0, 1, 0, 0xFF, 0xFF}, &got))
	assert.Equal(t, Packet{Version: 1, Screen: BroadcastIndex}, got)
}

func TestUnmarshalScreenControlIgnoresPayload(t *testing.T) {
	var got Packet
	require.NoError(t, Unmarshal([]byte{6, 0, 0, 2, 1, 0, 0xAA, 0xBB}, &got))
	assert.Equal(t, Packet{ScreenControl: true, Screen: 1}, got)
}

func TestUnmarshalIgnoresTrailingBytes(t *testing.T) {
	in := append(append([]byte{}, testPacketBytes...), 0x01, 0x02, 0x03)
	var got Packet
	require.NoError(t, Unmarshal(in, &got))
	assert.Equal(t, testPacket, got)
}

func TestUnmarshalReplacesExistingContents(t *testing.T) {
	got := Packet{Version: 9, Unicode: true, Displays: []Display{{Index: 1}, {Index: 2}}}
	require.NoError(t, Unmarshal(testPacketBytes, &got))
	assert.Equal(t, testPacket, got)
}

func TestUnmarshalErrors(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want error
	}{
		{"nil", nil, ErrShortPacket},
		{"one byte", []byte{4}, ErrShortPacket},
		{"pbc below header", []byte{3, 0, 0, 0, 0}, ErrMalformed},
		{"pbc exceeds buffer", []byte{5, 0, 0, 0, 0, 0}, ErrShortPacket},
		{"truncated display header", []byte{6, 0, 0, 0, 0, 0, 1, 0}, ErrMalformed},
		{"missing text length", []byte{8, 0, 0, 0, 0, 0, 1, 0, 0, 0}, ErrMalformed},
		{"text length overruns packet", []byte{10, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0xFF, 0xFF}, ErrMalformed},
		{"text length overruns pbc", []byte{11, 0, 0, 0, 0, 0, 1, 0, 0, 0, 2, 0, 'a', 'b'}, ErrMalformed},
		{"odd utf-16 length", []byte{11, 0, 0, 1, 0, 0, 1, 0, 0, 0, 1, 0, 'a'}, ErrMalformed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Packet
			assert.ErrorIs(t, Unmarshal(tt.in, &got), tt.want)
		})
	}
}
