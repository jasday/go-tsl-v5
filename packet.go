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
)

var (
	// ErrInvalidSize is when the provided byte slice does not contain enough control bits
	ErrInvalidSize = errors.New("invalid size")

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

// Unmarshal parses a byte slice containing a TSL v5 packet into p.
func Unmarshal(buffer []byte, p *Packet) error {
	packetSize := binary.LittleEndian.Uint16(buffer[0:2])
	if packetSize < 6 {
		return fmt.Errorf("%w: found length %d, should be at least %d", ErrInvalidSize, packetSize, 6)
	}

	if len(buffer) < int(packetSize) {
		return fmt.Errorf("buffer size is inconsistent with provided packet size")
	}

	p.Version = buffer[2]
	p.Screen = binary.LittleEndian.Uint16(buffer[4:6])
	p.Unicode = buffer[3] == 0x01
	p.ScreenControl = buffer[3] == 0x02

	if !p.ScreenControl {
		ptr := 6
		for {
			if ptr > int(packetSize) || ptr >= MaxUDPPacketSize-4 {
				break
			}

			d, newPtr := parseDisplay(buffer, ptr, p.Unicode)
			if d != nil {
				p.Displays = append(p.Displays, *d)
			}

			ptr = newPtr
		}
	}

	return nil
}

func parseDisplay(buffer []byte, start int, unicode bool) (*Display, int) {
	control := binary.LittleEndian.Uint16(buffer[start+2 : start+4])

	d := Display{
		Index:      binary.LittleEndian.Uint16(buffer[start : start+2]),
		RightTally: Lamp(control & 3),
		TextTally:  Lamp((control >> 2) & 3),
		LeftTally:  Lamp((control >> 4) & 3),
		Brightness: uint8((control >> 6) & 3),
	}

	length := 0
	if control&0x8000 == 0 {
		length = int(binary.LittleEndian.Uint16(buffer[start+4 : start+6]))
		data := buffer[start+6 : start+6+length]
		if unicode {
			if length%2 != 0 {
				return nil, 0
			}
			u := make([]uint16, length/2)
			for i := 0; i < length; i += 2 {
				u[i/2] = (uint16(data[i]) << 8) | uint16(data[i+1])
			}
			d.Text = string(utf16.Decode(u))
		} else {
			d.Text = string(data)
		}
	}
	return &d, start + 6 + length
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
		flags |= 1
	}
	if p.ScreenControl {
		flags |= 2
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
