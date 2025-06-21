package tally

import (
	"bytes"
	"encoding/binary"
	"unicode/utf16"

	"github.com/jasday/go-tsl-v5/pkg/display"
)

const (
	MaximumPacketSize int    = 2048
	packetControlData int    = 6 // 6 bytes of control data
	BroadcastIndex    uint16 = 0xFFFF
)

type Tally struct {
	pbc             uint16
	Version         byte
	Flags           Flags
	Screen          uint16
	DisplayMessages []display.Message
}

type Flags struct {
	UnicodeStrings bool
	ControlData    bool
}

func FromBuffer(buffer []byte) *Tally {
	packetSize := binary.LittleEndian.Uint16(buffer[0:2])
	if packetSize < uint16(packetControlData) {
		return &Tally{}
	}

	tally := Tally{
		pbc:     packetSize,
		Version: buffer[2],
		Screen:  binary.LittleEndian.Uint16(buffer[4:6]),
		Flags: Flags{
			UnicodeStrings: buffer[3] == 0x01,
			ControlData:    buffer[3] == 0x02,
		},
	}

	// If control data flag is cleared, next data is display message.
	// If set, data is screen control (not yet defined)
	if !tally.Flags.ControlData {
		ptr := 6
		for {
			if ptr > int(tally.pbc) || ptr >= MaximumPacketSize-4 {
				break
			}

			msg, newPtr := parseDisplayMessage(buffer, ptr, tally.Flags.UnicodeStrings)
			if msg != nil {
				tally.DisplayMessages = append(tally.DisplayMessages, *msg)
			}

			ptr = newPtr
		}
	}
	return &tally
}

func parseDisplayMessage(buffer []byte, startIndex int, unicodeStrings bool) (*display.Message, int) {
	controlFlags := binary.LittleEndian.Uint16(buffer[startIndex+2 : startIndex+4])

	msg := display.Message{
		Index: binary.LittleEndian.Uint16(buffer[startIndex : startIndex+2]),
		Control: display.Control{
			RightTally: parseTallyLampState(uint8(controlFlags) & 3),
			TextTally:  parseTallyLampState((uint8(controlFlags) & 12) >> 2),
			LeftTally:  parseTallyLampState((uint8(controlFlags) & 48) >> 4),
			Brightness: (uint8(controlFlags) & 192) >> 6,
		},
	}

	// If bit 15 is cleared, data is display text, else control info (not yet defined)
	if controlFlags&32768 != 32768 {
		msg.Data.Length = binary.LittleEndian.Uint16(buffer[startIndex+4 : startIndex+6])
		data := buffer[startIndex+6 : startIndex+6+int(msg.Data.Length)]
		if unicodeStrings {
			if msg.Data.Length%2 != 0 {
				// error, should be divisible by 2, don't try to decode this
				return nil, 0
			}
			u := make([]uint16, msg.Data.Length/2)
			for i := 0; i < int(msg.Data.Length); i += 2 {
				u[i/2] = (uint16(data[i]) << 8) | uint16(data[i+1])
			}
			msg.Data.Text = string(utf16.Decode(u))
		} else {
			msg.Data.Text = string(data)
		}
	}
	// Set the start index of the next display message
	return &msg, startIndex + 6 + int(msg.Data.Length)
}

func (t *Tally) Bytes(buffer []byte) []byte {
	// Reserve the first two bits for the PBC, set later.
	buffer = append(buffer, t.Version)

	flags := byte(0)
	if t.Flags.UnicodeStrings {
		flags += 1
	}
	if t.Flags.ControlData {
		flags += 2
	}
	buffer = append(buffer, flags)
	buffer = append(buffer, convertUint16ToUint8(t.Screen)...)

	for _, msg := range t.DisplayMessages {
		buffer = append(buffer, convertUint16ToUint8(msg.Index)...)

		tf := uint8(0)
		tf += uint8(msg.Control.RightTally)
		tf += (uint8(msg.Control.TextTally) << 2)
		tf += (uint8(msg.Control.LeftTally) << 4)
		tf += (uint8(msg.Control.Brightness) << 6)
		buffer = append(buffer, tf)

		if msg.Control.ControlData {
			buffer = append(buffer, uint8(128))
		} else {
			buffer = append(buffer, 0)
			txt := []byte(msg.Data.Text)
			buffer = append(buffer, convertUint16ToUint8(uint16(len(txt)))...)
			buffer = append(buffer, txt...)
		}

	}
	pcb := convertUint16ToUint8(uint16(len(buffer) - 2))
	buffer[0] = pcb[0]
	buffer[1] = pcb[1]
	return buffer
}

func parseTallyLampState(input uint8) display.Lamp {
	if input > 3 {
		return display.Off
	}
	return display.Lamp(input)
}

func convertUint16ToUint8(input uint16) []uint8 {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, []uint16{input})
	return []byte{buf.Bytes()[0], buf.Bytes()[1]}
}
