package tsl

import (
	"encoding"
	"encoding/binary"
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

func TestUnicodeText(t *testing.T) {
	p := Packet{Unicode: true, Displays: []Display{{Index: 1, Text: "Hé😀"}}}
	want := []byte{
		0x12, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x00, 0x08, 0x00,
		'H', 0x00, 0xE9, 0x00, 0x3D, 0xD8, 0x00, 0xDE, // UTF-16LE, surrogate pair for U+1F600
	}

	got, err := Marshal(&p)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	var decoded Packet
	require.NoError(t, Unmarshal(want, &decoded))
	assert.Equal(t, p, decoded)
}

func TestMarshalEncodesFields(t *testing.T) {
	p := Packet{
		Version: 1,
		Screen:  0x1234,
		Displays: []Display{
			{Index: BroadcastIndex, RightTally: LampRed, TextTally: LampGreen, LeftTally: LampAmber, Brightness: 2, Text: "AB"},
			{Index: 5, ControlData: true},
			{Index: 6},
		},
	}
	want := []byte{
		0x16, 0x00, 0x01, 0x00, 0x34, 0x12,
		0xFF, 0xFF, 0b10_11_10_01, 0x00, 0x02, 0x00, 'A', 'B',
		0x05, 0x00, 0x00, 0x80,
		0x06, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	got, err := Marshal(&p)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestMarshalScreenControl(t *testing.T) {
	got, err := Marshal(&Packet{ScreenControl: true, Screen: BroadcastIndex})
	require.NoError(t, err)
	assert.Equal(t, []byte{4, 0, 0, 2, 0xFF, 0xFF}, got)
}

func TestMarshalLampAndBrightnessBits(t *testing.T) {
	for _, l := range []Lamp{LampOff, LampRed, LampGreen, LampAmber} {
		for br := uint8(0); br <= 3; br++ {
			p := Packet{Displays: []Display{{RightTally: l, TextTally: l, LeftTally: l, Brightness: br}}}
			b, err := Marshal(&p)
			require.NoError(t, err)
			assert.Equal(t, byte(l)|byte(l)<<2|byte(l)<<4|br<<6, b[8])
			assert.Equal(t, byte(0), b[9])

			var got Packet
			require.NoError(t, Unmarshal(b, &got))
			assert.Equal(t, p, got)
		}
	}
}

func TestMarshalErrors(t *testing.T) {
	tests := []struct {
		name string
		p    *Packet
		want error
	}{
		{"nil", nil, ErrInvalidValue},
		{"right tally", &Packet{Displays: []Display{{RightTally: 4}}}, ErrInvalidValue},
		{"text tally", &Packet{Displays: []Display{{TextTally: 4}}}, ErrInvalidValue},
		{"left tally", &Packet{Displays: []Display{{LeftTally: 4}}}, ErrInvalidValue},
		{"brightness", &Packet{Displays: []Display{{Brightness: 4}}}, ErrInvalidValue},
		{"control data with text", &Packet{Displays: []Display{{ControlData: true, Text: "x"}}}, ErrInvalidValue},
		{"screen control with displays", &Packet{ScreenControl: true, Displays: []Display{{}}}, ErrInvalidValue},
		{"non-ASCII text", &Packet{Displays: []Display{{Text: "é"}}}, ErrInvalidValue},
		{"text too long", &Packet{Displays: []Display{{Text: string(make([]byte, 0x10000))}}}, ErrPacketTooLarge},
		{"packet too long", &Packet{Displays: []Display{{Text: string(make([]byte, 0xFFFF))}}}, ErrPacketTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Marshal(tt.p)
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

var (
	_ encoding.BinaryAppender    = (*Packet)(nil)
	_ encoding.BinaryMarshaler   = (*Packet)(nil)
	_ encoding.BinaryUnmarshaler = (*Packet)(nil)
)

func TestBinaryInterfaces(t *testing.T) {
	b, err := testPacket.MarshalBinary()
	require.NoError(t, err)
	assert.Equal(t, testPacketBytes, b)

	prefix := []byte{0xAA}
	b, err = testPacket.AppendBinary(prefix)
	require.NoError(t, err)
	assert.Equal(t, append([]byte{0xAA}, testPacketBytes...), b)

	var got Packet
	require.NoError(t, got.UnmarshalBinary(testPacketBytes))
	assert.Equal(t, testPacket, got)
}

func TestAppendBinaryDoesNotAllocate(t *testing.T) {
	buf := make([]byte, 0, 64)
	allocs := testing.AllocsPerRun(100, func() {
		_, _ = testPacket.AppendBinary(buf[:0])
	})
	assert.Zero(t, allocs)
}

func TestMarshalAllowsPacketsOverUDPLimit(t *testing.T) {
	b, err := Marshal(&Packet{Displays: []Display{{Text: string(make([]byte, 3000))}}})
	require.NoError(t, err)
	assert.Len(t, b, 6+6+3000)
}

func TestMarshalUDPSingle(t *testing.T) {
	got, err := MarshalUDP(&testPacket)
	require.NoError(t, err)
	assert.Equal(t, [][]byte{testPacketBytes}, got)
}

func TestMarshalUDPHeaderOnly(t *testing.T) {
	got, err := MarshalUDP(&Packet{ScreenControl: true})
	require.NoError(t, err)
	assert.Equal(t, [][]byte{{4, 0, 0, 2, 0, 0}}, got)
}

func TestMarshalUDPSplits(t *testing.T) {
	// Each display is 6+94 = 100 bytes, so 20 fit after the 6-byte header.
	text := string(make([]byte, 94))
	p := Packet{Version: 1, Unicode: false, Screen: 7}
	for i := range 45 {
		p.Displays = append(p.Displays, Display{Index: uint16(i), Brightness: 3, Text: text})
	}

	packets, err := MarshalUDP(&p)
	require.NoError(t, err)
	require.Len(t, packets, 3)

	var all []Display
	for i, b := range packets {
		assert.LessOrEqual(t, len(b), MaxUDPPacketSize)
		var got Packet
		require.NoError(t, Unmarshal(b, &got))
		assert.Equal(t, 2+int(binary.LittleEndian.Uint16(b)), len(b), "packet %d byte count", i)
		assert.Equal(t, p.Version, got.Version)
		assert.Equal(t, p.Screen, got.Screen)
		all = append(all, got.Displays...)
	}
	assert.Len(t, packets[0], 6+20*100)
	assert.Equal(t, p.Displays, all)
}

func TestMarshalUDPErrors(t *testing.T) {
	tests := []struct {
		name string
		p    *Packet
		want error
	}{
		{"nil", nil, ErrInvalidValue},
		{"invalid display", &Packet{Displays: []Display{{Brightness: 9}}}, ErrInvalidValue},
		{"screen control with displays", &Packet{ScreenControl: true, Displays: []Display{{}}}, ErrInvalidValue},
		{"display too large", &Packet{Displays: []Display{{Text: string(make([]byte, MaxUDPPacketSize))}}}, ErrPacketTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MarshalUDP(tt.p)
			assert.ErrorIs(t, err, tt.want)
		})
	}
}
