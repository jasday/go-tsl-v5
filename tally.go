package tsl

import (
	"bytes"
	"encoding/binary"
	"unicode/utf16"

	"github.com/jasday/go-tsl-v5/tally"
)

const (
	// MaximumPacketSize is the largest a TSL message can be when sent over UDP
	MaximumPacketSize int = 2048
	// BroadcastIndex is the broadcast index, sent to all displays
	BroadcastIndex uint16 = 0xFFFF
	// packetControlData is the minimum amount of data a valid tally message will contain
	packetControlData int = 6 // 6 bytes of control data
)

// Tally is the representation of a TSL tally
type Tally struct {
	Version         byte
	Flags           Flags
	Screen          uint16
	DisplayMessages []tally.Message
	pbc             uint16
}

// Flags is the representation of the control flags in a tally message
type Flags struct {
	UnicodeStrings bool
	ControlData    bool
}

func parseDisplayMessage(buffer []byte, startIndex int, unicodeStrings bool) (*tally.Message, int) {
	controlFlags := binary.LittleEndian.Uint16(buffer[startIndex+2 : startIndex+4])

	msg := tally.Message{
		Index: binary.LittleEndian.Uint16(buffer[startIndex : startIndex+2]),
		Control: tally.Control{
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

func parseTallyLampState(input uint8) tally.Lamp {
	if input > 3 {
		return tally.Off
	}
	return tally.Lamp(input)
}

func convertUint16ToUint8(input uint16) []uint8 {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, []uint16{input})
	return []byte{buf.Bytes()[0], buf.Bytes()[1]}
}
