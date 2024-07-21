package tally

import (
	"encoding/binary"
	"unicode/utf16"

	"github.com/jasday/go-tsl-v5/pkg/display"
)

const (
	maximumPacketSize int    = 2048
	packetControlData int    = 6 // 6 bytes of control data
	BroadcastIndex    uint16 = 0xFFFF
)

type Tally struct {
	Pbc             uint16
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
		return nil
	}

	tally := Tally{
		Pbc:     packetSize,
		Version: buffer[2],
		Screen:  binary.LittleEndian.Uint16(buffer[4:6]),
	}

	tally.Flags.UnicodeStrings = buffer[3] == 0x01
	tally.Flags.ControlData = buffer[3] == 0x02

	// If control data flag is cleared, next data is display message.
	// If set, data is screen control (not yet defined)
	if !tally.Flags.ControlData {
		ptr := 6
		for {
			controlFlags := binary.LittleEndian.Uint16(buffer[ptr+2 : ptr+4])
			msg := display.Message{
				Index: binary.LittleEndian.Uint16(buffer[ptr : ptr+2]),
				Control: display.Control{
					RightTally: parseTallyLampState(uint8(controlFlags) & 3),
					TextTally:  parseTallyLampState((uint8(controlFlags) & 12) >> 2),
					LeftTally:  parseTallyLampState((uint8(controlFlags) & 48) >> 4),
					Brightness: (uint8(controlFlags) & 192) >> 6,
				},
			}

			// If bit 15 is cleared, data is display text, else if control info (not yet defined)
			if controlFlags&32768 != 32768 {
				msg.Data.Length = binary.LittleEndian.Uint16(buffer[ptr+4 : ptr+6])
				data := buffer[ptr+6 : ptr+int(msg.Data.Length)]
				if tally.Flags.UnicodeStrings {
					if msg.Data.Length%2 != 0 {
						// error, should be divisible by 0
						// Do we just skip this one?
						break
					}
					shorts := make([]uint16, msg.Data.Length/2)
					for i := 0; i < int(msg.Data.Length); i += 2 {
						shorts[i/2] = (uint16(data[i]) << 8) | uint16(data[i+1])
					}
					msg.Data.Text = string(utf16.Decode(shorts))
				} else {
					msg.Data.Text = string(data)
				}
			}

			tally.DisplayMessages = append(tally.DisplayMessages, msg)
			ptr = ptr + 6 + int(msg.Data.Length)
		}
	}

	return &tally
}

func ToBuffer(buffer [maximumPacketSize]byte, tally Tally) {

}

func parseTallyLampState(input uint8) display.Tally {
	if input > 3 {
		return display.Off
	}
	return display.Tally(input)
}
