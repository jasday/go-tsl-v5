package tsl

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func FuzzUnmarshal(f *testing.F) {
	f.Add(testPacketBytes)
	f.Add([]byte{4, 0, 0, 3, 0, 0})
	f.Add([]byte{0x12, 0x00, 0x00, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x08, 0x00, 'H', 0x00, 0xE9, 0x00, 0x3D, 0xD8, 0x00, 0xDE})
	f.Add([]byte{8, 0, 0, 0, 0, 0, 5, 0, 0, 0x80})
	f.Fuzz(func(t *testing.T, in []byte) {
		var p Packet
		if Unmarshal(in, &p) != nil {
			return
		}
		// Decoded packets must re-encode. Non-ASCII bytes in ASCII mode and
		// invalid UTF-16 are not preserved, so only the decoded value is compared.
		if !p.Unicode && !isASCII(p) {
			return
		}
		out, err := Marshal(&p)
		require.NoError(t, err)
		var again Packet
		require.NoError(t, Unmarshal(out, &again))
		assert.Equal(t, p, again)
	})
}

func FuzzDecoder(f *testing.F) {
	f.Add(append([]byte{dle, stx}, testPacketBytes...))
	f.Add([]byte{dle, dle, stx, dle, stx, dle})
	f.Fuzz(func(t *testing.T, in []byte) {
		dec := NewDecoder(bytes.NewReader(in))
		// Every Decode consumes at least one byte, so the input must run out.
		for range len(in) + 1 {
			var p Packet
			err := dec.Decode(&p)
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return
			}
		}
		t.Fatal("decoder did not reach end of input")
	})
}

func isASCII(p Packet) bool {
	for _, d := range p.Displays {
		for i := range len(d.Text) {
			if d.Text[i] > 0x7F {
				return false
			}
		}
	}
	return true
}
