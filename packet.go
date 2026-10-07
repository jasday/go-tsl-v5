package tsl

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf16"
)

const (
	// MaxUDPPacketSize is the largest a TSL message can be when sent over UDP.
	MaxUDPPacketSize = 2048
	// BroadcastIndex addresses all screens or all displays.
	BroadcastIndex uint16 = 0xFFFF
	// headerSize is the size of VER, FLAGS and SCREEN.
	headerSize = 4

	flagUnicode       = 1 << 0
	flagScreenControl = 1 << 1
	controlDataBit    = 1 << 15
)

var (
	// ErrShortPacket is returned when a buffer is shorter than the packet it declares.
	ErrShortPacket = errors.New("tsl: short packet")

	// ErrMalformed is returned when a packet's contents are inconsistent.
	ErrMalformed = errors.New("tsl: malformed packet")

	// ErrInvalidValue is returned when a Packet cannot be encoded as given.
	ErrInvalidValue = errors.New("tsl: invalid value")

	// ErrPacketTooLarge is returned when a packet or text exceeds a 16-bit byte count.
	ErrPacketTooLarge = errors.New("tsl: packet too large")

	// ErrExceededMaximumPacket is returned when the resultant byte slice is too long for a UDP packet.
	ErrExceededMaximumPacket = errors.New("tally has exceeded maximum UDP packet size")
)

// Packet is a TSL v5 packet: a header followed by display messages.
type Packet struct {
	// Version is the minor protocol version (0 for v5.00).
	Version uint8
	// Unicode marks text as UTF-16LE rather than ASCII (FLAGS bit 0).
	Unicode bool
	// ScreenControl marks the packet as screen control data (FLAGS bit 1).
	ScreenControl bool
	// Screen is the screen index. BroadcastIndex addresses all screens.
	Screen uint16
	// Displays are the display messages (DMSG) in the packet.
	Displays []Display
}

// Display is a display message (DMSG) addressed to a single display.
type Display struct {
	// Index is the display index. BroadcastIndex addresses all displays.
	Index      uint16
	RightTally Lamp
	TextTally  Lamp
	LeftTally  Lamp
	// Brightness is in the range 0-3.
	Brightness uint8
	// ControlData marks the message as control data rather than text (CONTROL bit 15).
	ControlData bool
	// Text is the display text, encoded according to Packet.Unicode.
	Text string
}

// Unmarshal parses the TSL v5 packet at the start of buffer into p.
// Bytes after the packet are ignored. Reserved bits are ignored.
// Any existing contents of p are replaced.
func Unmarshal(buffer []byte, p *Packet) error {
	if len(buffer) < 2 {
		return fmt.Errorf("%w: need 2 bytes for byte count, have %d", ErrShortPacket, len(buffer))
	}
	pbc := int(binary.LittleEndian.Uint16(buffer))
	if pbc < headerSize {
		return fmt.Errorf("%w: byte count %d is less than header size %d", ErrMalformed, pbc, headerSize)
	}
	if len(buffer)-2 < pbc {
		return fmt.Errorf("%w: byte count %d exceeds remaining %d bytes", ErrShortPacket, pbc, len(buffer)-2)
	}
	data := buffer[2 : 2+pbc]

	*p = Packet{
		Version:       data[0],
		Unicode:       data[1]&flagUnicode != 0,
		ScreenControl: data[1]&flagScreenControl != 0,
		Screen:        binary.LittleEndian.Uint16(data[2:]),
	}

	// Screen control data is undefined in v5.0, so the rest of the packet is ignored.
	if p.ScreenControl {
		return nil
	}

	for off := headerSize; off < len(data); {
		d, n, err := parseDisplay(data[off:], p.Unicode)
		if err != nil {
			return fmt.Errorf("display message %d at offset %d: %w", len(p.Displays), off+2, err)
		}
		p.Displays = append(p.Displays, d)
		off += n
	}
	return nil
}

// parseDisplay parses the display message at the start of b and returns the number of bytes it used.
func parseDisplay(b []byte, unicode bool) (Display, int, error) {
	if len(b) < 4 {
		return Display{}, 0, fmt.Errorf("%w: %d bytes left, need 4 for index and control", ErrMalformed, len(b))
	}
	control := binary.LittleEndian.Uint16(b[2:])
	d := Display{
		Index:       binary.LittleEndian.Uint16(b),
		RightTally:  Lamp(control & 3),
		TextTally:   Lamp(control >> 2 & 3),
		LeftTally:   Lamp(control >> 4 & 3),
		Brightness:  uint8(control >> 6 & 3),
		ControlData: control&controlDataBit != 0,
	}
	// Control data is undefined in v5.0 and carries no payload.
	if d.ControlData {
		return d, 4, nil
	}

	if len(b) < 6 {
		return Display{}, 0, fmt.Errorf("%w: %d bytes left, need 6 for text length", ErrMalformed, len(b))
	}
	length := int(binary.LittleEndian.Uint16(b[4:]))
	if len(b)-6 < length {
		return Display{}, 0, fmt.Errorf("%w: text length %d exceeds remaining %d bytes", ErrMalformed, length, len(b)-6)
	}
	text, err := decodeText(b[6:6+length], unicode)
	if err != nil {
		return Display{}, 0, err
	}
	d.Text = text
	return d, 6 + length, nil
}

func decodeText(b []byte, unicode bool) (string, error) {
	if !unicode {
		return string(b), nil
	}
	if len(b)%2 != 0 {
		return "", fmt.Errorf("%w: odd UTF-16 text length %d", ErrMalformed, len(b))
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[2*i:])
	}
	return string(utf16.Decode(u)), nil
}

// appendText appends s as ASCII or UTF-16LE.
func appendText(b []byte, s string, unicode bool) []byte {
	if !unicode {
		return append(b, s...)
	}
	for _, u := range utf16.Encode([]rune(s)) {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return b
}

// Marshal encodes p as a TSL v5 packet.
// If the size exceeds the maximum UDP packet size, it returns the bytes with ErrExceededMaximumPacket.
func Marshal(p *Packet) ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("%w: nil packet", ErrInvalidValue)
	}
	b, err := p.appendTo(nil)
	if err != nil {
		return nil, err
	}
	if len(b) > MaxUDPPacketSize {
		return b, ErrExceededMaximumPacket
	}
	return b, nil
}

func (p *Packet) appendTo(b []byte) ([]byte, error) {
	if p.ScreenControl && len(p.Displays) > 0 {
		return nil, fmt.Errorf("%w: screen control packet cannot contain displays", ErrInvalidValue)
	}

	start := len(b)
	// The PBC is filled in once the packet length is known.
	b = append(b, 0, 0, p.Version)

	flags := byte(0)
	if p.Unicode {
		flags |= flagUnicode
	}
	if p.ScreenControl {
		flags |= flagScreenControl
	}
	b = append(b, flags)
	b = binary.LittleEndian.AppendUint16(b, p.Screen)

	for i := range p.Displays {
		var err error
		if b, err = p.Displays[i].appendTo(b, p.Unicode); err != nil {
			return nil, fmt.Errorf("display message %d: %w", i, err)
		}
	}

	pbc := len(b) - start - 2
	if pbc > 0xFFFF {
		return nil, fmt.Errorf("%w: byte count %d exceeds 65535", ErrPacketTooLarge, pbc)
	}
	binary.LittleEndian.PutUint16(b[start:], uint16(pbc))
	return b, nil
}

func (d *Display) appendTo(b []byte, unicode bool) ([]byte, error) {
	for _, l := range []Lamp{d.RightTally, d.TextTally, d.LeftTally} {
		if l > LampAmber {
			return nil, fmt.Errorf("%w: %v", ErrInvalidValue, l)
		}
	}
	if d.Brightness > 3 {
		return nil, fmt.Errorf("%w: brightness %d is greater than 3", ErrInvalidValue, d.Brightness)
	}

	control := uint16(d.RightTally) | uint16(d.TextTally)<<2 | uint16(d.LeftTally)<<4 | uint16(d.Brightness)<<6
	if d.ControlData {
		if d.Text != "" {
			return nil, fmt.Errorf("%w: control data message cannot contain text", ErrInvalidValue)
		}
		control |= controlDataBit
	}
	b = binary.LittleEndian.AppendUint16(b, d.Index)
	b = binary.LittleEndian.AppendUint16(b, control)
	if d.ControlData {
		return b, nil
	}

	if !unicode {
		for i := 0; i < len(d.Text); i++ {
			if d.Text[i] > 0x7F {
				return nil, fmt.Errorf("%w: non-ASCII text %q; set Packet.Unicode", ErrInvalidValue, d.Text)
			}
		}
	}

	lengthAt := len(b)
	b = appendText(append(b, 0, 0), d.Text, unicode)
	length := len(b) - lengthAt - 2
	if length > 0xFFFF {
		return nil, fmt.Errorf("%w: text length %d exceeds 65535", ErrPacketTooLarge, length)
	}
	binary.LittleEndian.PutUint16(b[lengthAt:], uint16(length))
	return b, nil
}
