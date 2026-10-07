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
		u[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
	}
	return string(utf16.Decode(u)), nil
}

// Marshal encodes p as a TSL v5 packet.
// If the size exceeds the maximum UDP packet size, it returns the bytes with ErrExceededMaximumPacket.
func Marshal(p *Packet) ([]byte, error) {
	if p == nil {
		return nil, errors.New("nil packet provided")
	}
	// Reserve the first two bytes for the PBC, set later.
	buffer := make([]byte, 2, MaxUDPPacketSize)
	buffer = append(buffer, p.Version)

	flags := byte(0)
	if p.Unicode {
		flags |= flagUnicode
	}
	if p.ScreenControl {
		flags |= flagScreenControl
	}
	buffer = append(buffer, flags)
	buffer = binary.LittleEndian.AppendUint16(buffer, p.Screen)

	for _, d := range p.Displays {
		buffer = binary.LittleEndian.AppendUint16(buffer, d.Index)

		tf := uint8(d.RightTally) | uint8(d.TextTally)<<2 | uint8(d.LeftTally)<<4 | d.Brightness<<6
		buffer = append(buffer, tf)

		if d.ControlData {
			buffer = append(buffer, 0x80)
		} else {
			buffer = append(buffer, 0)
			txt := []byte(d.Text)
			buffer = binary.LittleEndian.AppendUint16(buffer, uint16(len(txt)))
			buffer = append(buffer, txt...)
		}
	}

	binary.LittleEndian.PutUint16(buffer, uint16(len(buffer)-2))

	if len(buffer) > MaxUDPPacketSize {
		return buffer, ErrExceededMaximumPacket
	}
	return buffer, nil
}
