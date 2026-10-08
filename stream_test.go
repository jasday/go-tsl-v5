package tsl

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encodeFrame(t *testing.T, p *Packet) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, NewEncoder(&buf).Encode(p))
	return buf.Bytes()
}

func TestEncodeFrame(t *testing.T) {
	got := encodeFrame(t, &testPacket)
	assert.Equal(t, append([]byte{dle, stx}, testPacketBytes...), got)
}

func TestEncodeStuffsEveryDLE(t *testing.T) {
	// PBC 254 (0xFE), screen 0xFEFE and text containing 0xFE all need stuffing.
	p := Packet{Screen: 0xFEFE, Unicode: true, Displays: []Display{{Index: 0x00FE, Text: string(make([]rune, 121)) + "þ"}}}
	raw, err := Marshal(&p)
	require.NoError(t, err)
	require.Equal(t, byte(dle), raw[0], "test needs PBC low byte to be DLE")

	want := []byte{dle, stx}
	for _, b := range raw {
		want = append(want, b)
		if b == dle {
			want = append(want, dle)
		}
	}
	assert.Equal(t, want, encodeFrame(t, &p))

	var got Packet
	require.NoError(t, NewDecoder(bytes.NewReader(want)).Decode(&got))
	assert.Equal(t, p, got)
}

func TestEncodeErrors(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	require.ErrorIs(t, enc.Encode(nil), ErrInvalidValue)
	require.ErrorIs(t, enc.Encode(&Packet{Displays: []Display{{Brightness: 4}}}), ErrInvalidValue)
	assert.Zero(t, buf.Len())

	assert.Error(t, NewEncoder(errWriter{}).Encode(&testPacket))
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestDecodeSequence(t *testing.T) {
	packets := []Packet{
		testPacket,
		{Version: 1, Screen: BroadcastIndex, Displays: []Display{{Index: 0xFEFE, RightTally: LampRed, Text: "Cam 1"}}},
		{ScreenControl: true},
		{Unicode: true, Displays: []Display{{Text: "Kamera þ"}, {Index: 2, ControlData: true}}},
	}
	var stream bytes.Buffer
	stream.Write([]byte{0x00, dle, 0x01, dle, dle, 0x33}) // noise before the first frame
	enc := NewEncoder(&stream)
	for i := range packets {
		require.NoError(t, enc.Encode(&packets[i]))
	}

	readers := map[string]func(io.Reader) io.Reader{
		"whole":    func(r io.Reader) io.Reader { return r },
		"one byte": iotest.OneByteReader,
		"half":     iotest.HalfReader,
	}
	for name, wrap := range readers {
		t.Run(name, func(t *testing.T) {
			dec := NewDecoder(wrap(bytes.NewReader(stream.Bytes())))
			for i := range packets {
				var got Packet
				require.NoError(t, dec.Decode(&got), "packet %d", i)
				assert.Equal(t, packets[i], got, "packet %d", i)
			}
			var got Packet
			assert.ErrorIs(t, dec.Decode(&got), io.EOF)
		})
	}
}

func TestDecodeEOF(t *testing.T) {
	frame := encodeFrame(t, &testPacket)
	tests := []struct {
		name string
		in   []byte
		want error
	}{
		{"empty", nil, io.EOF},
		{"noise only", []byte{1, 2, 3}, io.EOF},
		{"trailing DLE", []byte{1, dle}, io.EOF},
		{"start only", []byte{dle, stx}, io.ErrUnexpectedEOF},
		{"in PBC", frame[:3], io.ErrUnexpectedEOF},
		{"in body", frame[:len(frame)-1], io.ErrUnexpectedEOF},
		{"after stuffed DLE", []byte{dle, stx, dle}, io.ErrUnexpectedEOF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Packet
			assert.ErrorIs(t, NewDecoder(bytes.NewReader(tt.in)).Decode(&got), tt.want)
		})
	}
}

func TestDecodeReadError(t *testing.T) {
	readErr := errors.New("read failed")
	frame := encodeFrame(t, &testPacket)
	var got Packet
	r := io.MultiReader(bytes.NewReader(frame[:5]), iotest.ErrReader(readErr))
	assert.ErrorIs(t, NewDecoder(r).Decode(&got), readErr)
}

func TestDecodeRecoversFromInterruptedFrame(t *testing.T) {
	frame := encodeFrame(t, &testPacket)
	// A frame cut short by the start of the next one.
	in := append(append([]byte{}, frame[:8]...), frame...)

	dec := NewDecoder(bytes.NewReader(in))
	var got Packet
	require.ErrorIs(t, dec.Decode(&got), ErrMalformed)
	require.NoError(t, dec.Decode(&got))
	assert.Equal(t, testPacket, got)
}

func TestDecodeRecoversFromBadEscape(t *testing.T) {
	frame := encodeFrame(t, &testPacket)
	in := append([]byte{dle, stx, 0x05, dle, 0x07}, frame...)

	dec := NewDecoder(bytes.NewReader(in))
	var got Packet
	require.ErrorIs(t, dec.Decode(&got), ErrMalformed)
	require.NoError(t, dec.Decode(&got))
	assert.Equal(t, testPacket, got)
}

func TestDecodeRecoversFromMalformedPacket(t *testing.T) {
	// Valid framing around a packet whose text length overruns its PBC.
	bad := []byte{dle, stx, 8, 0, 0, 0, 0, 0, 1, 0, 0, 0}
	in := slices.Concat(bad, encodeFrame(t, &testPacket))

	dec := NewDecoder(bytes.NewReader(in))
	var got Packet
	require.ErrorIs(t, dec.Decode(&got), ErrMalformed)
	require.NoError(t, dec.Decode(&got))
	assert.Equal(t, testPacket, got)
}
