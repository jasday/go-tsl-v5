package tsl

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Start of Message sequence is DLE/STX
const (
	dle = 0xFE
	stx = 0x02
)

var (
	// ErrInvalidSize is when the provided byte slice does not contain enough control bits
	ErrInvalidSize = errors.New("invalid size")

	// ErrExceededMaximumPacket is returned when the resultant byte slice is too long for a UDP packet.
	ErrExceededMaximumPacket = errors.New("tally has exceeded maximum UDP packet size")
)

// ReadMessage reads the next complete TSL v5 message from an io.Reader stream.
// This is intended for stream-oriented transports like TCP or serial.
// It synchronizes on the start-of-frame byte and reads the full message.
// Returns the Message and any error encountered during reading or unmarshaling.
func ReadMessage(r *bufio.Reader) (tally *Tally, err error) {
	buf := make([]byte, 1)

Delimiter:
	for {
		// Scan for dle
		for {
			_, err := r.Read(buf)
			if err != nil {
				return nil, err
			}
			if buf[0] == dle {
				break
			}
		}

		// Whenever we are in this loop, the previous byte was DLE
		for {
			// Check if the second is STX
			_, err = r.Read(buf)
			if err != nil {
				return nil, err
			}

			if buf[0] == dle {
				continue
			}

			if buf[0] == stx {
				// found DLE STX, start of message
				break Delimiter
			}

			break
		}
	}

	length := make([]byte, 2)
	if n, err := io.ReadFull(r, length); err != nil || n == 0 {
		return nil, err
	}

	size := binary.LittleEndian.Uint16(length)

	buffer := make([]byte, size)
	if _, err := io.ReadFull(r, buffer); err != nil {
		return nil, err
	}

	result := make([]byte, 0, len(buffer))
	result = append(result, length...)
	for i := 0; i < len(buffer); i++ {
		b := buffer[i]
		// Check for a byte-stuffed DLE
		if b == dle && i+1 < len(buffer) && buffer[i+1] == dle {
			result = append(result, dle) // keep one DLE
			i++                          // skip the second DLE
		} else {
			result = append(result, b) // keep normal byte
		}
	}

	tally = &Tally{}
	if err := Unmarshal(result, tally); err != nil {
		return nil, err
	}

	return tally, nil
}

// WriteMessage encodes a Tally struct into a TSL v5 frame and writes it to an io.Writer.
// This is intended for stream-oriented transports like TCP or serial.
// It handles DLE byte-stuffing in the payload and writes DLE STX + length + payload.
func WriteMessage(w io.Writer, tally *Tally) error {
	payload, err := Marshal(tally)
	if err != nil {
		return err
	}

	stuffed := make([]byte, 0, MaximumPacketSize)
	for i, b := range payload {
		stuffed = append(stuffed, b)
		if i == 0 || i == 1 {
			continue // Skip stuffing for the payload size bytes
		}

		if b == dle {
			// Duplicate DLE
			stuffed = append(stuffed, dle)
		}
	}

	// 3. Build frame: DLE STX + 2-byte length (little-endian) + stuffed payload
	frame := make([]byte, 0, 2+len(stuffed))
	frame = append(frame, dle, stx)

	length := uint16(len(stuffed))
	frame = append(frame, byte(length), byte(length>>8)) // little-endian

	frame = append(frame, stuffed...)

	// 4. Write full frame to the writer
	_, err = w.Write(frame)
	return err
}

// Unmarshal parses a byte slice containing a TSL v5 frame into a Message struct.
// It expects the slice to contain exactly one complete frame.
// Returns an error if the frame is malformed, has a bad checksum, or is incomplete.
func Unmarshal(buffer []byte, tally *Tally) error {
	packetSize := binary.LittleEndian.Uint16(buffer[0:2])
	if packetSize < uint16(packetControlData) {
		return fmt.Errorf("%w: found length %d, should be at least %d", ErrInvalidSize, packetSize, packetControlData)
	}

	if len(buffer) < int(packetSize) {
		return fmt.Errorf("buffer size is inconsistent with provided packet size")
	}

	tally.pbc = packetSize
	tally.Version = buffer[2]
	tally.Screen = binary.LittleEndian.Uint16(buffer[4:6])
	tally.Flags = Flags{
		UnicodeStrings: buffer[3] == 0x01,
		ControlData:    buffer[3] == 0x02,
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

	return nil
}

// Marshal converts a Message struct into a TSL v5-compliant byte slice.
// The resulting byte slice can be sent over TCP, serial, or UDP.
// Returns an error if the message is invalid or cannot be encoded.
// If the size exceeds the maximum UDP packet size, it will return an ExceededMaximumPacket error with the byte array
func Marshal(tally *Tally) ([]byte, error) {
	// Create a start buffer with len 2 to account for size insertion at end
	buffer := make([]byte, 2, MaximumPacketSize)
	if tally == nil {
		return nil, errors.New("nil tally provided")
	}

	// Reserve the first two bits for the PBC, set later.
	buffer = append(buffer, tally.Version)

	flags := byte(0)
	if tally.Flags.UnicodeStrings {
		flags += 1
	}

	if tally.Flags.ControlData {
		flags += 2
	}

	buffer = append(buffer, flags)
	buffer = append(buffer, convertUint16ToUint8(tally.Screen)...)

	for _, msg := range tally.DisplayMessages {
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

	if len(buffer) > MaximumPacketSize {
		return buffer, ErrExceededMaximumPacket
	}

	return buffer, nil
}
